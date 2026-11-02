package repo

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	logger "github.com/sirupsen/logrus"
)

// SyncResult holds the outcome of syncing a single repository.
type SyncResult struct {
	Name   string
	Status string
}

// RunSync syncs all repositories under rootDir in parallel.
func RunSync(rootDir string, runner GitRunner, output io.Writer) error {
	log := NewLogger(output)

	repos := FindAllRepos(rootDir)
	total := len(repos)
	if total == 0 {
		log.WithField("dir", rootDir).Warn("no git repositories found")
		return nil
	}

	workers := runtime.NumCPU()
	log.WithField("count", total).Info("found repositories to sync")

	sem := make(chan struct{}, workers)
	results := make([]SyncResult, total)
	var wg sync.WaitGroup

	for i, repoPath := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			result := SyncSingleRepo(path, rootDir, runner)
			log.WithFields(logger.Fields{
				logFieldRepo:   result.Name,
				logFieldStatus: result.Status,
			}).Info("sync result")
			results[idx] = result
		}(i, repoPath)
	}

	wg.Wait()

	synced, wip, failed := 0, 0, 0
	for _, r := range results {
		switch {
		case strings.HasPrefix(r.Status, "synced"):
			synced++
			if strings.Contains(r.Status, "wip") {
				wip++
			}
		case strings.HasPrefix(r.Status, "FAIL"):
			failed++
		}
	}

	log.WithFields(logger.Fields{
		statusSynced:       synced,
		"wip":              wip,
		mirrorStatusFailed: failed,
	}).Info("summary")
	return nil
}

// SyncSingleRepo syncs a single repository with fetch/rebase and WIP preservation.
func SyncSingleRepo(repoPath, rootDir string, runner GitRunner) SyncResult {
	name := strings.TrimPrefix(repoPath, rootDir+"/")

	// Fetch before touching anything: a failure leaves the repository as it was found, and
	// the default branch is then detected from what the remote reports today.
	if err := runner.Run(repoPath, "fetch", "--all", "--prune", "-q"); err != nil {
		return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (fetch: %v)", err)}
	}
	refreshRemoteHEAD(repoPath, originRemote, runner)

	defaultBranch, detectErr := DetectDefaultBranch(repoPath, runner)
	if detectErr != nil {
		return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (%v)", detectErr)}
	}

	currentBranch := DetectCurrentBranch(repoPath, defaultBranch, runner)
	isDirty := runner.Output(repoPath, "status", "--porcelain") != ""
	wipBranch := fmt.Sprintf("wip/%s", currentBranch)

	if isDirty {
		if result, ok := SaveWIPState(repoPath, name, currentBranch, wipBranch, runner); !ok {
			return result
		}
	}

	return SyncAndRestore(repoPath, name, defaultBranch, currentBranch, wipBranch, isDirty, runner)
}

// DetectCurrentBranch detects the currently checked out branch.
func DetectCurrentBranch(repoPath, defaultBranch string, runner GitRunner) string {
	branch := runner.Output(repoPath, "branch", "--show-current")
	if branch == "" {
		return defaultBranch
	}
	return branch
}

// SaveWIPState creates a WIP branch and commits all changes. Returns (result, ok).
func SaveWIPState(repoPath, name, currentBranch, wipBranch string, runner GitRunner) (SyncResult, bool) {
	if err := runner.Run(repoPath, "checkout", "-B", wipBranch); err != nil {
		return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (wip branch: %v)", err)}, false
	}

	if err := runner.Run(repoPath, "add", "-A"); err != nil {
		_ = runner.Run(repoPath, "checkout", currentBranch)
		_ = runner.Run(repoPath, "branch", "-D", wipBranch)
		return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (wip add: %v)", err)}, false
	}

	msg := fmt.Sprintf("wip: auto-stash %s", time.Now().Format("2006-01-02T15:04:05"))
	if err := runner.Run(repoPath, "commit", "--no-verify", "-m", msg); err != nil {
		_ = runner.Run(repoPath, "checkout", currentBranch)
		_ = runner.Run(repoPath, "branch", "-D", wipBranch)
		return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (wip commit: %v)", err)}, false
	}

	return SyncResult{}, true
}

// SyncAndRestore checks out the default branch, rebases it onto its upstream, and restores
// the original branch state. The repository must already be fetched.
func SyncAndRestore(
	repoPath, name, defaultBranch, currentBranch, wipBranch string, isDirty bool, runner GitRunner,
) SyncResult {
	if err := checkoutDefaultBranch(repoPath, defaultBranch, runner); err != nil {
		RestoreBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
		return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (checkout %s: %v)", defaultBranch, err)}
	}

	if err := runner.Run(repoPath, "pull", "--rebase"); err != nil {
		_ = runner.Run(repoPath, "rebase", "--abort")
		RestoreBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
		return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (pull --rebase: %v)", err)}
	}

	return FinalizeSync(repoPath, name, defaultBranch, wipBranch, isDirty, runner)
}

// checkoutDefaultBranch switches to the default branch, creating it from origin when it was
// never checked out here, as happens after the remote renames its default. The remote is
// named explicitly because a bare checkout cannot choose when another remote, such as a
// fork's upstream, carries a branch with the same name.
func checkoutDefaultBranch(repoPath, defaultBranch string, runner GitRunner) error {
	if refExists(repoPath, localBranchRefPrefix+defaultBranch, runner) {
		return runner.Run(repoPath, "checkout", defaultBranch)
	}
	return runner.Run(repoPath, "checkout", "--track", originRemote+"/"+defaultBranch)
}

// RestoreBranch restores the appropriate branch after a failure.
func RestoreBranch(repoPath, currentBranch, wipBranch string, isDirty bool, runner GitRunner) {
	if isDirty {
		_ = runner.Run(repoPath, "checkout", wipBranch)
	} else {
		_ = runner.Run(repoPath, "checkout", currentBranch)
	}
}

// FinalizeSync rebases the WIP branch (if dirty) and ensures the repo ends on the default branch.
func FinalizeSync(
	repoPath, name, defaultBranch, wipBranch string, isDirty bool, runner GitRunner,
) SyncResult {
	status := statusSynced
	if isDirty {
		_ = runner.Run(repoPath, "checkout", wipBranch)
		if err := runner.Run(repoPath, "rebase", defaultBranch); err != nil {
			_ = runner.Run(repoPath, "rebase", "--abort")
		}
		if err := runner.Run(repoPath, "checkout", defaultBranch); err != nil {
			return SyncResult{Name: name, Status: fmt.Sprintf("FAIL (checkout %s: %v)", defaultBranch, err)}
		}
		status = fmt.Sprintf("synced (wip: %s)", wipBranch)
	}
	return SyncResult{Name: name, Status: status}
}
