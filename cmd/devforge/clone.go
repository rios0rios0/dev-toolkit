package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	adoProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/azuredevops"
	ghProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/github"
	glProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/gitlab"
	gitRegistry "github.com/rios0rios0/gitforge/pkg/registry/infrastructure"
	"github.com/spf13/cobra"
)

const (
	scanDepthNested = 2
	maxCloneArgs    = 2
	splitOwnerLimit = 2
	sshFailCode     = 255
	dirPermissions  = 0o750
)

//nolint:gochecknoglobals // read-only configuration lookup table
var providerPathMap = map[string]string{
	"github.com":    "github",
	"dev.azure.com": "azuredevops",
	"gitlab.com":    "gitlab",
}

//nolint:gochecknoglobals // read-only configuration lookup table
var providerScanDepth = map[string]int{
	"github":      1,
	"azuredevops": scanDepthNested,
	"gitlab":      1,
}

//nolint:gochecknoglobals // read-only configuration lookup table
var providerTokenEnv = map[string]string{
	"github":      "GH_TOKEN",
	"azuredevops": "AZURE_DEVOPS_EXT_PAT",
	"gitlab":      "GITLAB_TOKEN",
}

//nolint:gochecknoglobals // read-only configuration lookup table
var providerHostMap = map[string]string{
	"github":      "github.com",
	"azuredevops": "dev.azure.com",
	"gitlab":      "gitlab.com",
}

func newCloneCmd() *cobra.Command {
	var dryRun bool
	var includeArchived bool

	cmd := &cobra.Command{
		Use:   "clone <ssh-alias> [root-dir]",
		Short: "Clone missing repositories from a Git provider",
		Long: `Discovers repositories from the Git provider, compares with local directories,
clones missing repos via SSH, and optionally removes extra local repos.`,
		Args: cobra.RangeArgs(1, maxCloneArgs),
		RunE: func(_ *cobra.Command, args []string) error {
			sshAlias := args[0]
			rootDir, _ := os.Getwd()
			if len(args) > 1 {
				rootDir = args[1]
			}
			rootDir = filepath.Clean(rootDir)
			return runClone(rootDir, sshAlias, dryRun, includeArchived)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
	cmd.Flags().BoolVar(&includeArchived, "include-archived", false, "include archived repositories")

	return cmd
}

func runClone(rootDir, sshAlias string, dryRun, includeArchived bool) error {
	providerName, owner, err := detectProviderAndOwner(rootDir)
	if err != nil {
		return err
	}

	logf("provider=%s owner=%s", providerName, owner)
	if dryRun {
		logf("(dry-run mode)")
	}

	provider, resolveErr := resolveProvider(providerName)
	if resolveErr != nil {
		return resolveErr
	}

	remoteRepos, discoverErr := discoverRepos(provider, owner, includeArchived)
	if discoverErr != nil {
		return discoverErr
	}

	depth := providerScanDepth[providerName]
	localRepos := scanLocalRepos(rootDir, depth)
	logf("found %d local repositories", len(localRepos))

	missing, extra := computeDiff(remoteRepos, localRepos)
	logf("%d missing, %d extra", len(missing), len(extra))

	if len(missing) == 0 && len(extra) == 0 {
		logf("everything is in sync")
		return nil
	}

	cloned, failed := cloneMissing(missing, providerName, sshAlias, rootDir, dryRun)
	handleExtraRepos(extra, rootDir, dryRun)

	logf("summary: %d cloned, %d failed, %d extra", cloned, failed, len(extra))
	return nil
}

func resolveProvider(providerName string) (globalEntities.ForgeProvider, error) {
	envVar := providerTokenEnv[providerName]
	token := os.Getenv(envVar)
	if token == "" {
		return nil, fmt.Errorf("%s environment variable not set", envVar)
	}

	registry := newProviderRegistry()
	provider, err := registry.Get(providerName, token)
	if err != nil {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
	return provider, nil
}

func discoverRepos(
	provider globalEntities.ForgeProvider, owner string, includeArchived bool,
) ([]globalEntities.Repository, error) {
	logf("discovering remote repositories...")
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

	logf("found %d remote repositories", len(remoteRepos))
	return remoteRepos, nil
}

func computeDiff(
	remoteRepos []globalEntities.Repository, localRepos []string,
) ([]globalEntities.Repository, []string) {
	remoteSet := make(map[string]globalEntities.Repository, len(remoteRepos))
	for _, r := range remoteRepos {
		remoteSet[repoKey(r)] = r
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

func cloneMissing(
	missing []globalEntities.Repository,
	providerName, sshAlias, rootDir string,
	dryRun bool,
) (int, int) {
	if len(missing) == 0 {
		return 0, 0
	}

	if dryRun {
		for _, r := range missing {
			url := sshCloneURL(r, providerName, sshAlias)
			target := filepath.Join(rootDir, repoKey(r))
			logf("would clone %s -> %s", url, target)
		}
		return 0, 0
	}

	if preflightErr := sshPreflight(providerName, sshAlias); preflightErr != nil {
		logf("ERROR: %v", preflightErr)
		return 0, len(missing)
	}

	return parallelClone(missing, providerName, sshAlias, rootDir)
}

func handleExtraRepos(extra []string, rootDir string, dryRun bool) {
	if len(extra) == 0 {
		return
	}

	isInteractive := false
	if fi, statErr := os.Stdin.Stat(); statErr == nil {
		isInteractive = fi.Mode()&os.ModeCharDevice != 0
	}

	for _, name := range extra {
		switch {
		case dryRun:
			logf("extra: %s", name)
		case !isInteractive:
			logf("extra: %s (kept, non-interactive)", name)
		default:
			promptDeleteExtra(name, rootDir)
		}
	}
}

func promptDeleteExtra(name, rootDir string) {
	fmt.Fprintf(os.Stderr, "[dev] \"%s\" exists locally but not on remote. Delete? [y/N] ", name)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() && strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
		if removeErr := os.RemoveAll(filepath.Join(rootDir, name)); removeErr != nil {
			logf("ERROR: could not delete %s: %v", name, removeErr)
		} else {
			logf("deleted %s", name)
		}
	} else {
		logf("kept %s", name)
	}
}

func detectProviderAndOwner(rootDir string) (string, string, error) {
	for pathSegment, name := range providerPathMap {
		idx := strings.Index(rootDir, "/"+pathSegment+"/")
		if idx < 0 {
			continue
		}
		after := rootDir[idx+len("/"+pathSegment+"/"):]
		parts := strings.SplitN(after, "/", splitOwnerLimit)
		if parts[0] == "" {
			return "", "", fmt.Errorf("could not extract owner from path: %s", rootDir)
		}
		return name, parts[0], nil
	}
	return "", "", fmt.Errorf("could not detect provider from path: %s", rootDir)
}

func newProviderRegistry() *gitRegistry.ProviderRegistry {
	r := gitRegistry.NewProviderRegistry()
	r.RegisterFactory("github", ghProvider.NewProvider)
	r.RegisterFactory("azuredevops", adoProvider.NewProvider)
	r.RegisterFactory("gitlab", glProvider.NewProvider)
	return r
}

func repoKey(r globalEntities.Repository) string {
	if r.Project != "" {
		return r.Project + "/" + r.Name
	}
	return r.Name
}

func scanLocalRepos(rootDir string, depth int) []string {
	if depth == scanDepthNested {
		return scanNestedRepos(rootDir)
	}
	return scanFlatRepos(rootDir)
}

func scanFlatRepos(rootDir string) []string {
	var repos []string
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return repos
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitDir := filepath.Join(rootDir, e.Name(), ".git")
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			repos = append(repos, e.Name())
		}
	}
	return repos
}

func scanNestedRepos(rootDir string) []string {
	var repos []string
	projects, err := os.ReadDir(rootDir)
	if err != nil {
		return repos
	}
	for _, p := range projects {
		if !p.IsDir() {
			continue
		}
		repos = append(repos, scanProjectRepos(rootDir, p.Name())...)
	}
	return repos
}

func scanProjectRepos(rootDir, projectName string) []string {
	var repos []string
	subEntries, err := os.ReadDir(filepath.Join(rootDir, projectName))
	if err != nil {
		return repos
	}
	for _, e := range subEntries {
		if !e.IsDir() {
			continue
		}
		gitDir := filepath.Join(rootDir, projectName, e.Name(), ".git")
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			repos = append(repos, projectName+"/"+e.Name())
		}
	}
	return repos
}

func sshPreflight(providerName, sshAlias string) error {
	host, ok := providerHostMap[providerName]
	if !ok {
		return fmt.Errorf("unknown provider for SSH preflight: %s", providerName)
	}
	sshHost := fmt.Sprintf("%s-%s", host, sshAlias)
	logf("verifying SSH connectivity to %s...", sshHost)

	cmd := exec.CommandContext(
		context.Background(), "ssh", "-T", "-o", "ConnectTimeout=10",
		fmt.Sprintf("git@%s", sshHost),
	) // #nosec G204
	cmd.Stdin = nil
	err := cmd.Run()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == sshFailCode {
		return fmt.Errorf("SSH connection to %s failed (check SSH config and keys)", sshHost)
	}
	logf("SSH connectivity OK")
	return nil
}

type cloneResult struct {
	name    string
	success bool
	err     string
}

func parallelClone(
	repos []globalEntities.Repository,
	providerName, sshAlias, rootDir string,
) (int, int) {
	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([]cloneResult, len(repos))
	var wg sync.WaitGroup

	logf("cloning %d repos (%d parallel workers)", len(repos), workers)

	for i, r := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, repo globalEntities.Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = cloneSingleRepo(repo, providerName, sshAlias, rootDir)
		}(i, r)
	}

	wg.Wait()

	cloned, failed := 0, 0
	for _, r := range results {
		if r.success {
			fmt.Fprintf(os.Stderr, "  %-50s CLONED\n", r.name)
			cloned++
		} else {
			fmt.Fprintf(os.Stderr, "  %-50s FAIL (%s)\n", r.name, r.err)
			failed++
		}
	}
	return cloned, failed
}

func cloneSingleRepo(
	repo globalEntities.Repository,
	providerName, sshAlias, rootDir string,
) cloneResult {
	url := sshCloneURL(repo, providerName, sshAlias)
	target := filepath.Join(rootDir, repoKey(repo))

	if mkdirErr := os.MkdirAll(filepath.Dir(target), dirPermissions); mkdirErr != nil {
		return cloneResult{name: repoKey(repo), err: mkdirErr.Error()}
	}

	cmd := exec.CommandContext(
		context.Background(), "git", "clone", url, target,
	) // #nosec G204
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(),
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new -o BatchMode=yes",
	)
	output, cloneErr := cmd.CombinedOutput()
	if cloneErr != nil {
		return cloneResult{name: repoKey(repo), err: strings.TrimSpace(string(output))}
	}
	return cloneResult{name: repoKey(repo), success: true}
}

func sshCloneURL(repo globalEntities.Repository, providerName, sshAlias string) string {
	host := providerHostMap[providerName]
	aliasHost := fmt.Sprintf("%s-%s", host, sshAlias)
	if repo.SSHURL != "" {
		return strings.Replace(repo.SSHURL, host, aliasHost, 1)
	}
	if repo.Project != "" {
		return fmt.Sprintf(
			"git@%s:v3/%s/%s/%s", aliasHost, repo.Organization, repo.Project, repo.Name,
		)
	}
	return fmt.Sprintf("git@%s:%s/%s.git", aliasHost, repo.Organization, repo.Name)
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[dev] "+format+"\n", args...)
}
