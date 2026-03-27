package repo

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// RestoreConfig holds all dependencies for a restore operation.
type RestoreConfig struct {
	RootDir string
	Runner  GitRunner
	Output  logger.FieldLogger
}

// RestoreResult holds the outcome of restoring a single repository.
type RestoreResult struct {
	Name   string
	Status string
}

// RunRestore restores GitHub as the primary origin after a failover.
func RunRestore(cfg RestoreConfig) error {
	log := cfg.Output

	repos := FindAllRepos(cfg.RootDir)
	if len(repos) == 0 {
		log.WithField("dir", cfg.RootDir).Warn("no git repositories found")
		return nil
	}

	log.WithField("count", len(repos)).Info("starting restore")

	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([]RestoreResult, len(repos))
	var wg sync.WaitGroup

	for i, repoPath := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = restoreSingleRepo(path, cfg.RootDir, cfg.Runner)
		}(i, repoPath)
	}

	wg.Wait()

	restoreStatusCategory := map[string]string{
		"restored":                        "restored",
		"skipped (not in failover state)": "skipped",
	}

	counts := map[string]int{"restored": 0, "skipped": 0, "failed": 0}
	for _, r := range results {
		log.WithFields(logger.Fields{
			"repo":   r.Name,
			"status": r.Status,
		}).Info(r.Status)
		category, known := restoreStatusCategory[r.Status]
		if !known {
			category = "failed"
		}
		counts[category]++
	}

	log.WithFields(logger.Fields{
		"restored": counts["restored"],
		"skipped":  counts["skipped"],
		"failed":   counts["failed"],
	}).Info("restore completed")
	return nil
}

func restoreSingleRepo(repoPath, rootDir string, runner GitRunner) RestoreResult {
	name, _ := filepath.Rel(rootDir, repoPath)

	// check if in failover state (github remote exists as backup)
	githubURL := runner.Output(repoPath, "remote", "get-url", "github")
	if githubURL == "" {
		return RestoreResult{Name: name, Status: "skipped (not in failover state)"}
	}

	// push any Codeberg-only commits back to GitHub
	pushErr := runner.Run(repoPath, "push", "github", "--all", "--tags")

	// rename origin -> codeberg
	if err := runner.Run(repoPath, "remote", "rename", "origin", "codeberg"); err != nil {
		return RestoreResult{Name: name, Status: fmt.Sprintf("FAIL (rename origin: %v)", err)}
	}

	// rename github -> origin
	if err := runner.Run(repoPath, "remote", "rename", "github", "origin"); err != nil {
		// rollback
		_ = runner.Run(repoPath, "remote", "rename", "codeberg", "origin")
		return RestoreResult{Name: name, Status: fmt.Sprintf("FAIL (rename github: %v)", err)}
	}

	if pushErr != nil {
		return RestoreResult{Name: name, Status: fmt.Sprintf("restored (push failed: %v)", pushErr)}
	}
	return RestoreResult{Name: name, Status: "restored"}
}
