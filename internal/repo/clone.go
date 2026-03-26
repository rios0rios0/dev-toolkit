package repo

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
)

const maxCloneArgs = 2

// PreflightFunc is a function that verifies SSH connectivity before cloning.
type PreflightFunc func(providerName, sshAlias string, output io.Writer) error

// CloneConfig holds all dependencies for a clone operation.
type CloneConfig struct {
	RootDir         string
	SSHAlias        string
	DryRun          bool
	IncludeArchived bool
	Provider        globalEntities.ForgeProvider
	Runner          GitRunner
	Output          io.Writer
	Input           io.Reader
	Preflight       PreflightFunc
}

// RunClone executes the full clone workflow.
func RunClone(cfg CloneConfig) error {
	providerName, owner, err := DetectProviderAndOwner(cfg.RootDir)
	if err != nil {
		return err
	}

	Logf(cfg.Output, "provider=%s owner=%s", providerName, owner)
	if cfg.DryRun {
		Logf(cfg.Output, "(dry-run mode)")
	}

	remoteRepos, discoverErr := DiscoverRepos(cfg.Provider, owner, cfg.IncludeArchived, cfg.Output)
	if discoverErr != nil {
		return discoverErr
	}

	depth := ProviderScanDepth(providerName)
	localRepos := ScanLocalRepos(cfg.RootDir, depth)
	Logf(cfg.Output, "found %d local repositories", len(localRepos))

	missing, extra := ComputeDiff(remoteRepos, localRepos)
	Logf(cfg.Output, "%d missing, %d extra", len(missing), len(extra))

	if len(missing) == 0 && len(extra) == 0 {
		Logf(cfg.Output, "everything is in sync")
		return nil
	}

	cloned, failed := CloneMissing(missing, cfg)
	HandleExtraRepos(extra, cfg.RootDir, cfg.DryRun, cfg.Input, cfg.Output)

	Logf(cfg.Output, "summary: %d cloned, %d failed, %d extra", cloned, failed, len(extra))
	return nil
}

// DiscoverRepos fetches repositories from the provider and optionally filters archived ones.
func DiscoverRepos(
	provider globalEntities.ForgeProvider, owner string, includeArchived bool, output io.Writer,
) ([]globalEntities.Repository, error) {
	Logf(output, "discovering remote repositories...")
	remoteRepos, err := provider.DiscoverRepositories(context.Background(), owner)
	if err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}

	if !includeArchived {
		var filtered []globalEntities.Repository
		for _, r := range remoteRepos {
			if !r.IsArchived {
				filtered = append(filtered, r)
			}
		}
		remoteRepos = filtered
	}

	Logf(output, "found %d remote repositories", len(remoteRepos))
	return remoteRepos, nil
}

// ComputeDiff computes the missing and extra repositories between remote and local sets.
func ComputeDiff(
	remoteRepos []globalEntities.Repository, localRepos []string,
) ([]globalEntities.Repository, []string) {
	remoteSet := make(map[string]globalEntities.Repository, len(remoteRepos))
	for _, r := range remoteRepos {
		remoteSet[Key(r)] = r
	}

	localSet := make(map[string]struct{}, len(localRepos))
	for _, name := range localRepos {
		localSet[name] = struct{}{}
	}

	var missing []globalEntities.Repository
	for key, r := range remoteSet {
		if _, ok := localSet[key]; !ok {
			missing = append(missing, r)
		}
	}

	var extra []string
	for _, name := range localRepos {
		if _, ok := remoteSet[name]; !ok {
			extra = append(extra, name)
		}
	}

	return missing, extra
}

// CloneMissing clones missing repositories, respecting dry-run mode.
func CloneMissing(missing []globalEntities.Repository, cfg CloneConfig) (int, int) {
	if len(missing) == 0 {
		return 0, 0
	}

	if cfg.DryRun {
		for _, r := range missing {
			url := cfg.Provider.SSHCloneURL(r, cfg.SSHAlias)
			target := filepath.Join(cfg.RootDir, Key(r))
			Logf(cfg.Output, "would clone %s -> %s", url, target)
		}
		return 0, 0
	}

	preflight := cfg.Preflight
	if preflight == nil {
		preflight = SSHPreflight
	}
	providerName, _, _ := DetectProviderAndOwner(cfg.RootDir)
	if preflightErr := preflight(providerName, cfg.SSHAlias, cfg.Output); preflightErr != nil {
		Logf(cfg.Output, "ERROR: %v", preflightErr)
		return 0, len(missing)
	}

	return ParallelClone(missing, cfg.Provider, cfg.SSHAlias, cfg.RootDir, cfg.Runner, cfg.Output)
}

// sshSuccessPatterns are stderr fragments that indicate the remote Git server
// responded, confirming that SSH connectivity and authentication succeeded.
// Different providers use different messages and exit codes (e.g., Azure DevOps
// exits 255 even on success), so we check the output rather than the exit code.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var sshSuccessPatterns = []string{
	"shell access is not supported",  // Azure DevOps
	"successfully authenticated",     // GitHub
	"welcome to gitlab",              // GitLab
}

// SSHPreflight verifies SSH connectivity to the provider host via the SSH config alias.
func SSHPreflight(providerName, sshAlias string, output io.Writer) error {
	host := ProviderHost(providerName)
	if host == "" {
		return fmt.Errorf("unknown provider for SSH preflight: %s", providerName)
	}

	sshHost := host
	if sshAlias != "" {
		sshHost = fmt.Sprintf("%s-%s", host, sshAlias)
	}
	Logf(output, "verifying SSH connectivity to %s...", sshHost)

	var stderr bytes.Buffer
	cmd := exec.CommandContext(
		context.Background(), "ssh", "-T", "-o", "ConnectTimeout=10",
		fmt.Sprintf("git@%s", sshHost),
	) // #nosec G204
	cmd.Stdin = nil
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		Logf(output, "SSH connectivity OK")
		return nil
	}

	stderrStr := stderr.String()
	stderrLower := strings.ToLower(stderrStr)
	for _, pattern := range sshSuccessPatterns {
		if strings.Contains(stderrLower, pattern) {
			Logf(output, "SSH connectivity OK")
			return nil
		}
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return fmt.Errorf("SSH preflight failed: %w", err)
	}

	return fmt.Errorf("SSH connection to %s failed (check SSH config and keys): %s",
		sshHost, strings.TrimSpace(stderrStr))
}

type cloneResult struct {
	name    string
	success bool
	err     string
}

// ParallelClone clones repositories in parallel using goroutines with a semaphore.
func ParallelClone(
	repos []globalEntities.Repository,
	provider globalEntities.ForgeProvider,
	sshAlias, rootDir string,
	runner GitRunner,
	output io.Writer,
) (int, int) {
	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([]cloneResult, len(repos))
	var wg sync.WaitGroup

	Logf(output, "cloning %d repos (%d parallel workers)", len(repos), workers)

	for i, r := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, repo globalEntities.Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = cloneSingleRepo(repo, provider, sshAlias, rootDir, runner)
		}(i, r)
	}

	wg.Wait()

	cloned, failed := 0, 0
	for _, r := range results {
		if r.success {
			fmt.Fprintf(output, "  %-50s CLONED\n", r.name)
			cloned++
		} else {
			fmt.Fprintf(output, "  %-50s FAIL (%s)\n", r.name, r.err)
			failed++
		}
	}
	return cloned, failed
}

func cloneSingleRepo(
	repo globalEntities.Repository,
	provider globalEntities.ForgeProvider,
	sshAlias, rootDir string,
	runner GitRunner,
) cloneResult {
	url := provider.SSHCloneURL(repo, sshAlias)
	target := filepath.Join(rootDir, Key(repo))

	if cloneErr := runner.Clone(url, target); cloneErr != nil {
		return cloneResult{name: Key(repo), err: cloneErr.Error()}
	}
	return cloneResult{name: Key(repo), success: true}
}

// HandleExtraRepos prompts for deletion of extra local repos or skips in non-interactive mode.
func HandleExtraRepos(extra []string, rootDir string, dryRun bool, input io.Reader, output io.Writer) {
	if len(extra) == 0 {
		return
	}

	isInteractive := isTerminal(input)

	for _, name := range extra {
		switch {
		case dryRun:
			Logf(output, "extra: %s", name)
		case !isInteractive:
			Logf(output, "extra: %s (kept, non-interactive)", name)
		default:
			PromptDeleteExtra(name, rootDir, input, output)
		}
	}
}

func isTerminal(input io.Reader) bool {
	f, ok := input.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// PromptDeleteExtra asks the user to confirm deletion of an extra local repo.
func PromptDeleteExtra(name, rootDir string, input io.Reader, output io.Writer) {
	fmt.Fprintf(output, "[dev] \"%s\" exists locally but not on remote. Delete? [y/N] ", name)
	scanner := bufio.NewScanner(input)
	if scanner.Scan() && strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
		if removeErr := os.RemoveAll(filepath.Join(rootDir, name)); removeErr != nil {
			Logf(output, "ERROR: could not delete %s: %v", name, removeErr)
		} else {
			Logf(output, "deleted %s", name)
		}
	} else {
		Logf(output, "kept %s", name)
	}
}

// MaxCloneArgs returns the maximum number of positional arguments for the clone command.
func MaxCloneArgs() int {
	return maxCloneArgs
}
