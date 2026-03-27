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
	logger "github.com/sirupsen/logrus"
)

const maxCloneArgs = 2

// PreflightFunc is a function that verifies SSH connectivity before cloning.
type PreflightFunc func(providerName, sshAlias string, log logger.FieldLogger) error

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
	Logger          logger.FieldLogger
}

func (c *CloneConfig) log() logger.FieldLogger {
	if c.Logger == nil {
		c.Logger = NewLogger(c.Output)
	}
	return c.Logger
}

// RunClone executes the full clone workflow.
func RunClone(cfg CloneConfig) error {
	log := cfg.log()

	providerName, owner, err := DetectProviderAndOwner(cfg.RootDir)
	if err != nil {
		return err
	}

	log.WithFields(logger.Fields{
		"provider": providerName,
		"owner":    owner,
	}).Info("clone workflow started")
	if cfg.DryRun {
		log.Info("dry-run mode enabled")
	}

	remoteRepos, discoverErr := DiscoverRepos(cfg.Provider, owner, cfg.IncludeArchived, log)
	if discoverErr != nil {
		return discoverErr
	}

	depth := ProviderScanDepth(providerName)
	localRepos := ScanLocalRepos(cfg.RootDir, depth)
	log.WithField("count", len(localRepos)).Info("scanned local repositories")

	missing, extra := ComputeDiff(remoteRepos, localRepos)
	log.WithFields(logger.Fields{
		"missing": len(missing),
		"extra":   len(extra),
	}).Info("computed repository diff")

	if len(missing) == 0 && len(extra) == 0 {
		log.Info("everything is in sync")
		return nil
	}

	cloned, failed := CloneMissing(missing, cfg)
	HandleExtraRepos(extra, cfg.RootDir, cfg.DryRun, cfg.Input, cfg.Output, log)

	log.WithFields(logger.Fields{
		"cloned": cloned,
		"failed": failed,
		"extra":  len(extra),
	}).Info("clone workflow completed")
	return nil
}

// DiscoverRepos fetches repositories from the provider and optionally filters archived ones.
func DiscoverRepos(
	provider globalEntities.ForgeProvider, owner string, includeArchived bool, log logger.FieldLogger,
) ([]globalEntities.Repository, error) {
	log.Info("discovering remote repositories")
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

	log.WithField("count", len(remoteRepos)).Info("remote repositories discovered")
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
	log := cfg.log()

	if len(missing) == 0 {
		return 0, 0
	}

	if cfg.DryRun {
		for _, r := range missing {
			url := cfg.Provider.SSHCloneURL(r, cfg.SSHAlias)
			target := filepath.Join(cfg.RootDir, Key(r))
			log.WithFields(logger.Fields{
				"repo":   Key(r),
				"url":    url,
				"target": target,
			}).Info("would clone repository")
		}
		return 0, 0
	}

	preflight := cfg.Preflight
	if preflight == nil {
		preflight = SSHPreflight
	}
	providerName, _, _ := DetectProviderAndOwner(cfg.RootDir)
	if preflightErr := preflight(providerName, cfg.SSHAlias, log); preflightErr != nil {
		log.WithError(preflightErr).Error("SSH preflight failed")
		return 0, len(missing)
	}

	return ParallelClone(missing, cfg.Provider, cfg.SSHAlias, cfg.RootDir, cfg.Runner, log)
}

// sshSuccessPatterns are stderr fragments that indicate the remote Git server
// responded, confirming that SSH connectivity and authentication succeeded.
// Different providers use different messages and exit codes (e.g., Azure DevOps
// exits 255 even on success), so we check the output rather than the exit code.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var sshSuccessPatterns = []string{
	"shell access is not supported", // Azure DevOps
	"successfully authenticated",    // GitHub
	"welcome to gitlab",             // GitLab
}

// IsSSHSuccess checks whether SSH stderr output contains a provider-specific
// success pattern, indicating that connectivity and authentication succeeded
// despite a non-zero exit code.
func IsSSHSuccess(stderr string) bool {
	lower := strings.ToLower(stderr)
	for _, pattern := range sshSuccessPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// SSHPreflight verifies SSH connectivity to the provider host via the SSH config alias.
func SSHPreflight(providerName, sshAlias string, log logger.FieldLogger) error {
	host := ProviderHost(providerName)
	if host == "" {
		return fmt.Errorf("unknown provider for SSH preflight: %s", providerName)
	}

	sshHost := host
	if sshAlias != "" {
		sshHost = fmt.Sprintf("%s-%s", host, sshAlias)
	}
	log.WithField("host", sshHost).Info("verifying SSH connectivity")

	var stderr bytes.Buffer
	cmd := exec.CommandContext(
		context.Background(),
		"ssh",
		"-T",
		"-o", "ConnectTimeout=10",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		fmt.Sprintf("git@%s", sshHost),
	) // #nosec G204
	cmd.Stdin = nil
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		log.WithField("host", sshHost).Info("SSH connectivity verified")
		return nil
	}

	stderrStr := stderr.String()
	if IsSSHSuccess(stderrStr) {
		log.WithField("host", sshHost).Info("SSH connectivity verified")
		return nil
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
	log logger.FieldLogger,
) (int, int) {
	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([]cloneResult, len(repos))
	var wg sync.WaitGroup

	log.WithFields(logger.Fields{
		"count":   len(repos),
		"workers": workers,
	}).Info("starting parallel clone")

	for i, r := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, repo globalEntities.Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = cloneSingleRepo(repo, provider, sshAlias, rootDir, runner, log)
		}(i, r)
	}

	wg.Wait()

	cloned, failed := 0, 0
	for _, r := range results {
		if r.success {
			cloned++
		} else {
			failed++
		}
	}

	log.WithFields(logger.Fields{
		"cloned": cloned,
		"failed": failed,
	}).Info("parallel clone completed")
	return cloned, failed
}

func cloneSingleRepo(
	repo globalEntities.Repository,
	provider globalEntities.ForgeProvider,
	sshAlias, rootDir string,
	runner GitRunner,
	log logger.FieldLogger,
) cloneResult {
	url := provider.SSHCloneURL(repo, sshAlias)
	target := filepath.Join(rootDir, Key(repo))
	repoKey := Key(repo)

	log.WithFields(logger.Fields{
		"repo":   repoKey,
		"url":    url,
		"target": target,
	}).Info("cloning repository")

	if cloneErr := runner.Clone(url, target); cloneErr != nil {
		log.WithFields(logger.Fields{
			"repo": repoKey,
		}).WithError(cloneErr).Error("clone failed")
		return cloneResult{name: repoKey, err: cloneErr.Error()}
	}

	log.WithFields(logger.Fields{
		"repo":   repoKey,
		"target": target,
	}).Info("repository cloned")
	return cloneResult{name: repoKey, success: true}
}

// HandleExtraRepos prompts for deletion of extra local repos or skips in non-interactive mode.
func HandleExtraRepos(
	extra []string, rootDir string, dryRun bool, input io.Reader, output io.Writer, log logger.FieldLogger,
) {
	if len(extra) == 0 {
		return
	}

	isInteractive := isTerminal(input)

	for _, name := range extra {
		switch {
		case dryRun:
			log.WithField("repo", name).Warn("extra repository")
		case !isInteractive:
			log.WithField("repo", name).Warn("extra repository (kept, non-interactive)")
		default:
			PromptDeleteExtra(name, rootDir, input, output, log)
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
func PromptDeleteExtra(name, rootDir string, input io.Reader, output io.Writer, log logger.FieldLogger) {
	fmt.Fprintf(output, "[dev] \"%s\" exists locally but not on remote. Delete? [y/N] ", name)
	scanner := bufio.NewScanner(input)
	if scanner.Scan() && strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
		if removeErr := os.RemoveAll(filepath.Join(rootDir, name)); removeErr != nil {
			log.WithFields(logger.Fields{
				"repo": name,
			}).WithError(removeErr).Error("could not delete repository")
		} else {
			log.WithField("repo", name).Info("deleted extra repository")
		}
	} else {
		log.WithField("repo", name).Info("kept extra repository")
	}
}

// MaxCloneArgs returns the maximum number of positional arguments for the clone command.
func MaxCloneArgs() int {
	return maxCloneArgs
}
