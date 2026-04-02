package repo

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// PruneResult holds the outcome of pruning a single repository.
type PruneResult struct {
	Name    string
	Deleted []string
	Status  string
}

// isPruneFailure returns true if the given status string indicates a failure.
func isPruneFailure(status string) bool {
	return strings.HasPrefix(status, "FAIL") || strings.Contains(strings.ToLower(status), "failed")
}

// RunPrune deletes merged branches in all repositories under rootDir in parallel.
func RunPrune(rootDir string, runner GitRunner, dryRun bool, output io.Writer) error {
	log := NewLogger(output)

	repos := ScanFlatRepos(rootDir)
	total := len(repos)
	if total == 0 {
		log.WithField("dir", rootDir).Warn("no git repositories found")
		return nil
	}

	workers := runtime.NumCPU()
	log.WithField("count", total).Info("found repositories to prune")
	if dryRun {
		log.Info("dry-run mode enabled")
	}

	sem := make(chan struct{}, workers)
	results := make([]PruneResult, total)
	var wg sync.WaitGroup

	for i, repoName := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, name string) {
			defer wg.Done()
			defer func() { <-sem }()
			repoPath := filepath.Join(rootDir, name)
			result := PruneSingleRepo(repoPath, rootDir, runner, dryRun)
			if len(result.Deleted) > 0 || isPruneFailure(result.Status) {
				log.WithFields(logger.Fields{
					"repo":   result.Name,
					"status": result.Status,
				}).Info("prune result")
			}
			results[idx] = result
		}(i, repoName)
	}

	wg.Wait()

	pruned, skipped, failed := 0, 0, 0
	for _, r := range results {
		switch {
		case isPruneFailure(r.Status):
			failed++
		case len(r.Deleted) > 0:
			pruned += len(r.Deleted)
		default:
			skipped++
		}
	}

	log.WithFields(logger.Fields{
		"pruned": pruned,
		"clean":  skipped,
		"failed": failed,
	}).Info("summary")
	return nil
}

// PruneSingleRepo finds and deletes local branches that have been merged into the default branch.
func PruneSingleRepo(repoPath, rootDir string, runner GitRunner, dryRun bool) PruneResult {
	name, _ := filepath.Rel(rootDir, repoPath)
	defaultBranch := DetectDefaultBranch(repoPath, runner)

	merged := ListMergedBranches(repoPath, defaultBranch, runner)
	if len(merged) == 0 {
		return PruneResult{Name: name, Status: "clean"}
	}

	if dryRun {
		status := fmt.Sprintf("would delete %d branches: %s", len(merged), strings.Join(merged, ", "))
		return PruneResult{Name: name, Deleted: merged, Status: status}
	}

	var deleted []string
	var errors []string
	for _, branch := range merged {
		if err := runner.Run(repoPath, "branch", "-d", branch); err != nil {
			errors = append(errors, fmt.Sprintf("%s (%v)", branch, err))
		} else {
			deleted = append(deleted, branch)
		}
	}

	if len(errors) > 0 {
		status := fmt.Sprintf("deleted %d, failed %d: %s", len(deleted), len(errors), strings.Join(errors, "; "))
		return PruneResult{Name: name, Deleted: deleted, Status: status}
	}

	return PruneResult{
		Name:    name,
		Deleted: deleted,
		Status:  fmt.Sprintf("deleted %d branches: %s", len(deleted), strings.Join(deleted, ", ")),
	}
}

// ListMergedBranches returns local branches that have been merged into the given base branch,
// excluding the base branch itself and any HEAD pointer.
func ListMergedBranches(repoPath, baseBranch string, runner GitRunner) []string {
	output := runner.Output(repoPath, "branch", "--merged", baseBranch)
	if output == "" {
		return nil
	}

	var merged []string
	for line := range strings.SplitSeq(output, "\n") {
		branch := strings.TrimSpace(line)
		branch = strings.TrimPrefix(branch, "* ")
		// skip empty lines, the base branch itself, and HEAD pointer/detached HEAD lines
		if branch == "" || branch == baseBranch || strings.HasPrefix(branch, "HEAD") {
			continue
		}
		merged = append(merged, branch)
	}
	return merged
}
