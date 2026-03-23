package repo

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
)

// PruneResult holds the outcome of pruning a single repository.
type PruneResult struct {
	Name    string
	Deleted []string
	Status  string
}

// RunPrune deletes merged branches in all repositories under rootDir in parallel.
func RunPrune(rootDir string, runner GitRunner, dryRun bool, output io.Writer) error {
	repos := ScanFlatRepos(rootDir)
	total := len(repos)
	if total == 0 {
		Logf(output, "no git repositories found in %s", rootDir)
		return nil
	}

	workers := runtime.NumCPU()
	Logf(output, "found %d repositories to prune", total)
	if dryRun {
		Logf(output, "(dry-run mode)")
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
			repoPath := rootDir + "/" + name
			results[idx] = PruneSingleRepo(repoPath, rootDir, runner, dryRun)
		}(i, repoName)
	}

	wg.Wait()

	pruned, skipped, failed := 0, 0, 0
	for _, r := range results {
		if len(r.Deleted) > 0 || strings.HasPrefix(r.Status, "FAIL") {
			Logf(output, "%s: %s", r.Name, r.Status)
		}
		switch {
		case strings.HasPrefix(r.Status, "FAIL"):
			failed++
		case len(r.Deleted) > 0:
			pruned += len(r.Deleted)
		default:
			skipped++
		}
	}

	Logf(output, "summary: %d branches pruned, %d repos clean, %d failed", pruned, skipped, failed)
	return nil
}

// PruneSingleRepo finds and deletes local branches that have been merged into the default branch.
func PruneSingleRepo(repoPath, rootDir string, runner GitRunner, dryRun bool) PruneResult {
	name := strings.TrimPrefix(repoPath, rootDir+"/")
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
	for _, line := range strings.Split(output, "\n") {
		branch := strings.TrimSpace(line)
		branch = strings.TrimPrefix(branch, "* ")
		if branch == "" || branch == baseBranch || strings.Contains(branch, "HEAD") {
			continue
		}
		merged = append(merged, branch)
	}
	return merged
}
