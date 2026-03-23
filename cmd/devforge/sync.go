package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [root-dir]",
		Short: "Sync all repositories under a directory",
		Long: `For each repository found under the root directory, fetches all remotes,
rebases the default branch, and preserves any uncommitted work via WIP commits.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			rootDir, _ := os.Getwd()
			if len(args) > 0 {
				rootDir = args[0]
			}
			rootDir = filepath.Clean(rootDir)
			return runSync(rootDir)
		},
	}
}

func runSync(rootDir string) error {
	repos := findAllRepos(rootDir)
	total := len(repos)
	if total == 0 {
		logf("no git repositories found in %s", rootDir)
		return nil
	}

	workers := runtime.NumCPU()
	logf("found %d repositories to sync", total)

	sem := make(chan struct{}, workers)
	results := make([]syncResult, total)
	var wg sync.WaitGroup

	for i, repoPath := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = syncSingleRepo(path, rootDir)
		}(i, repoPath)
	}

	wg.Wait()

	synced, wip, failed := 0, 0, 0
	for _, r := range results {
		logf("%s: %s", r.name, r.status)
		switch {
		case strings.HasPrefix(r.status, "synced"):
			synced++
			if strings.Contains(r.status, "wip") {
				wip++
			}
		case strings.HasPrefix(r.status, "FAIL"):
			failed++
		}
	}

	logf("summary: %d synced, %d with WIP commits, %d failed", synced, wip, failed)
	return nil
}

type syncResult struct {
	name   string
	status string
}

func syncSingleRepo(repoPath, rootDir string) syncResult {
	name := strings.TrimPrefix(repoPath, rootDir+"/")

	// detect default branch
	defaultBranch := gitOutput(repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	defaultBranch = strings.TrimPrefix(defaultBranch, "refs/remotes/origin/")
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// save current branch
	currentBranch := gitOutput(repoPath, "branch", "--show-current")
	if currentBranch == "" {
		currentBranch = defaultBranch
	}

	// check for dirty tree
	isDirty := gitOutput(repoPath, "status", "--porcelain") != ""
	wipBranch := fmt.Sprintf("wip/%s", currentBranch)

	if isDirty {
		logf("%s: dirty tree, saving to %s", name, wipBranch)
		// create or reset WIP branch and commit
		if err := gitRun(repoPath, "checkout", "-B", wipBranch); err != nil {
			return syncResult{name: name, status: fmt.Sprintf("FAIL (wip branch: %v)", err)}
		}
		if err := gitRun(repoPath, "add", "-A"); err != nil {
			_ = gitRun(repoPath, "checkout", currentBranch)
			_ = gitRun(repoPath, "branch", "-D", wipBranch)
			return syncResult{name: name, status: fmt.Sprintf("FAIL (wip add: %v)", err)}
		}
		msg := fmt.Sprintf("wip: auto-stash %s", time.Now().Format("2006-01-02T15:04:05"))
		if err := gitRun(repoPath, "commit", "--no-verify", "-m", msg); err != nil {
			_ = gitRun(repoPath, "checkout", currentBranch)
			_ = gitRun(repoPath, "branch", "-D", wipBranch)
			return syncResult{name: name, status: fmt.Sprintf("FAIL (wip commit: %v)", err)}
		}
	}

	// checkout default branch and sync
	if err := gitRun(repoPath, "checkout", defaultBranch); err != nil {
		if isDirty {
			_ = gitRun(repoPath, "checkout", wipBranch)
		}
		return syncResult{name: name, status: fmt.Sprintf("FAIL (checkout %s: %v)", defaultBranch, err)}
	}

	if err := gitRun(repoPath, "fetch", "--all", "--prune", "-q"); err != nil {
		if isDirty {
			_ = gitRun(repoPath, "checkout", wipBranch)
		} else {
			_ = gitRun(repoPath, "checkout", currentBranch)
		}
		return syncResult{name: name, status: fmt.Sprintf("FAIL (fetch: %v)", err)}
	}

	if err := gitRun(repoPath, "pull", "--rebase"); err != nil {
		_ = gitRun(repoPath, "rebase", "--abort")
		if isDirty {
			_ = gitRun(repoPath, "checkout", wipBranch)
		} else {
			_ = gitRun(repoPath, "checkout", currentBranch)
		}
		return syncResult{name: name, status: fmt.Sprintf("FAIL (pull --rebase: %v)", err)}
	}

	// restore original state
	status := "synced"
	if isDirty {
		_ = gitRun(repoPath, "checkout", wipBranch)
		if err := gitRun(repoPath, "rebase", defaultBranch); err != nil {
			_ = gitRun(repoPath, "rebase", "--abort")
		}
		_ = gitRun(repoPath, "checkout", currentBranch)
		status = fmt.Sprintf("synced (wip: %s)", wipBranch)
	} else if currentBranch != defaultBranch {
		_ = gitRun(repoPath, "checkout", currentBranch)
	}

	return syncResult{name: name, status: status}
}

func findAllRepos(rootDir string) []string {
	var repos []string
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			if repoPath != rootDir {
				repos = append(repos, repoPath)
			}
			return filepath.SkipDir
		}
		return nil
	})
	return repos
}

func gitRun(dir string, args ...string) error {
	cmd := exec.Command("git", args...) // #nosec G204
	cmd.Dir = dir
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return nil
}

func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...) // #nosec G204
	cmd.Dir = dir
	cmd.Stdin = nil
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
