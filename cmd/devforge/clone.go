package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	gitRegistry "github.com/rios0rios0/gitforge/pkg/registry/infrastructure"

	ghProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/github"
	adoProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/azuredevops"
	glProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/gitlab"

	"github.com/spf13/cobra"
)

var providerPathMap = map[string]string{
	"github.com":    "github",
	"dev.azure.com": "azuredevops",
	"gitlab.com":    "gitlab",
}

var providerScanDepth = map[string]int{
	"github":      1,
	"azuredevops": 2,
	"gitlab":      1,
}

var providerTokenEnv = map[string]string{
	"github":      "GH_TOKEN",
	"azuredevops": "AZURE_DEVOPS_EXT_PAT",
	"gitlab":      "GITLAB_TOKEN",
}

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
		Args: cobra.RangeArgs(1, 2),
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

	// resolve token
	envVar := providerTokenEnv[providerName]
	token := os.Getenv(envVar)
	if token == "" {
		return fmt.Errorf("%s environment variable not set", envVar)
	}

	// create provider and discover remote repos
	registry := newProviderRegistry()
	provider, err := registry.Get(providerName, token)
	if err != nil {
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	logf("discovering remote repositories...")
	remoteRepos, err := provider.DiscoverRepositories(context.Background(), owner)
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
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

	// scan local repos
	depth := providerScanDepth[providerName]
	localRepos := scanLocalRepos(rootDir, depth)
	logf("found %d local repositories", len(localRepos))

	// compute diff
	remoteSet := make(map[string]globalEntities.Repository, len(remoteRepos))
	for _, r := range remoteRepos {
		key := repoKey(r)
		remoteSet[key] = r
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

	logf("%d missing, %d extra", len(missing), len(extra))

	if len(missing) == 0 && len(extra) == 0 {
		logf("everything is in sync")
		return nil
	}

	// clone missing repos
	cloned, failed := 0, 0
	if len(missing) > 0 {
		if dryRun {
			for _, r := range missing {
				url := sshCloneURL(r, providerName, sshAlias)
				target := filepath.Join(rootDir, repoKey(r))
				logf("would clone %s -> %s", url, target)
			}
		} else {
			// SSH preflight
			if err := sshPreflight(providerName, sshAlias); err != nil {
				return err
			}

			cloned, failed = parallelClone(missing, providerName, sshAlias, rootDir)
		}
	}

	// handle extra repos
	isInteractive := false
	if fi, err := os.Stdin.Stat(); err == nil {
		isInteractive = fi.Mode()&os.ModeCharDevice != 0
	}
	for _, name := range extra {
		if dryRun {
			logf("extra: %s", name)
		} else if !isInteractive {
			logf("extra: %s (kept, non-interactive)", name)
		} else {
			fmt.Fprintf(os.Stderr, "[dev] \"%s\" exists locally but not on remote. Delete? [y/N] ", name)
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() && strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
				if err := os.RemoveAll(filepath.Join(rootDir, name)); err != nil {
					logf("ERROR: could not delete %s: %v", name, err)
				} else {
					logf("deleted %s", name)
				}
			} else {
				logf("kept %s", name)
			}
		}
	}

	logf("summary: %d cloned, %d failed, %d extra", cloned, failed, len(extra))
	return nil
}

func detectProviderAndOwner(rootDir string) (string, string, error) {
	for pathSegment, name := range providerPathMap {
		idx := strings.Index(rootDir, "/"+pathSegment+"/")
		if idx < 0 {
			continue
		}
		after := rootDir[idx+len("/"+pathSegment+"/"):]
		parts := strings.SplitN(after, "/", 2)
		if parts[0] == "" {
			return "", "", fmt.Errorf("could not extract owner from path: %s", rootDir)
		}
		return name, parts[0], nil
	}
	return "", "", fmt.Errorf("could not detect provider from path: %s", rootDir)
}

func newProviderRegistry() *gitRegistry.ProviderRegistry {
	r := gitRegistry.NewProviderRegistry()
	r.RegisterFactory("github", func(token string) globalEntities.ForgeProvider {
		return ghProvider.NewProvider(token)
	})
	r.RegisterFactory("azuredevops", func(token string) globalEntities.ForgeProvider {
		return adoProvider.NewProvider(token)
	})
	r.RegisterFactory("gitlab", func(token string) globalEntities.ForgeProvider {
		return glProvider.NewProvider(token)
	})
	return r
}

func repoKey(r globalEntities.Repository) string {
	if r.Project != "" {
		return r.Project + "/" + r.Name
	}
	return r.Name
}

func scanLocalRepos(rootDir string, depth int) []string {
	var repos []string
	if depth == 1 {
		entries, err := os.ReadDir(rootDir)
		if err != nil {
			return repos
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			gitDir := filepath.Join(rootDir, e.Name(), ".git")
			if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
				repos = append(repos, e.Name())
			}
		}
	} else if depth == 2 {
		projects, err := os.ReadDir(rootDir)
		if err != nil {
			return repos
		}
		for _, p := range projects {
			if !p.IsDir() {
				continue
			}
			subEntries, err := os.ReadDir(filepath.Join(rootDir, p.Name()))
			if err != nil {
				continue
			}
			for _, e := range subEntries {
				if !e.IsDir() {
					continue
				}
				gitDir := filepath.Join(rootDir, p.Name(), e.Name(), ".git")
				if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
					repos = append(repos, p.Name()+"/"+e.Name())
				}
			}
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

	cmd := exec.Command("ssh", "-T", "-o", "ConnectTimeout=10", fmt.Sprintf("git@%s", sshHost)) // #nosec G204
	cmd.Stdin = nil
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 255 {
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
) (cloned, failed int) {
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

			url := sshCloneURL(repo, providerName, sshAlias)
			target := filepath.Join(rootDir, repoKey(repo))

			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				results[idx] = cloneResult{name: repoKey(repo), err: err.Error()}
				return
			}

			cmd := exec.Command("git", "clone", url, target) // #nosec G204
			cmd.Stdin = nil
			cmd.Env = append(os.Environ(),
				"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new -o BatchMode=yes",
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				results[idx] = cloneResult{name: repoKey(repo), err: strings.TrimSpace(string(output))}
			} else {
				results[idx] = cloneResult{name: repoKey(repo), success: true}
			}
		}(i, r)
	}

	wg.Wait()

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

func sshCloneURL(repo globalEntities.Repository, providerName, sshAlias string) string {
	host := providerHostMap[providerName]
	aliasHost := fmt.Sprintf("%s-%s", host, sshAlias)
	if repo.SSHURL != "" {
		return strings.Replace(repo.SSHURL, host, aliasHost, 1)
	}
	if repo.Project != "" {
		return fmt.Sprintf("git@%s:v3/%s/%s/%s", aliasHost, repo.Organization, repo.Project, repo.Name)
	}
	return fmt.Sprintf("git@%s:%s/%s.git", aliasHost, repo.Organization, repo.Name)
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[dev] "+format+"\n", args...)
}
