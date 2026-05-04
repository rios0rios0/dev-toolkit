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
			result := restoreSingleRepo(path, cfg.RootDir, cfg.Runner)
			log.WithFields(logger.Fields{
				logFieldRepo:   result.Name,
				logFieldStatus: result.Status,
			}).Info(result.Status)
			results[idx] = result
		}(i, repoPath)
	}

	wg.Wait()

	restoreStatusCategory := map[string]string{
		statusRestored:                    statusRestored,
		"skipped (not in failover state)": statusSkipped,
	}

	counts := map[string]int{statusRestored: 0, statusSkipped: 0, mirrorStatusFailed: 0}
	for _, r := range results {
		category, known := restoreStatusCategory[r.Status]
		if !known {
			category = mirrorStatusFailed
		}
		counts[category]++
	}

	log.WithFields(logger.Fields{
		statusRestored:     counts[statusRestored],
		statusSkipped:      counts[statusSkipped],
		mirrorStatusFailed: counts[mirrorStatusFailed],
	}).Info("restore completed")
	return nil
}

func restoreSingleRepo(repoPath, rootDir string, runner GitRunner) RestoreResult {
	name, _ := filepath.Rel(rootDir, repoPath)

	// check if in failover state (github remote exists as backup)
	githubURL := runner.Output(repoPath, "remote", "get-url", ProviderGitHub)
	if githubURL == "" {
		return RestoreResult{Name: name, Status: "skipped (not in failover state)"}
	}

	// push any Codeberg-only commits back to GitHub
	pushErr := runner.Run(repoPath, "push", ProviderGitHub, "--all", "--tags")

	// rename origin -> codeberg
	if err := runner.Run(repoPath, "remote", "rename", "origin", ProviderCodeberg); err != nil {
		return RestoreResult{Name: name, Status: fmt.Sprintf("FAIL (rename origin: %v)", err)}
	}

	// rename github -> origin
	if err := runner.Run(repoPath, "remote", "rename", ProviderGitHub, "origin"); err != nil {
		// rollback
		_ = runner.Run(repoPath, "remote", "rename", ProviderCodeberg, "origin")
		return RestoreResult{Name: name, Status: fmt.Sprintf("FAIL (rename github: %v)", err)}
	}

	if pushErr != nil {
		return RestoreResult{Name: name, Status: fmt.Sprintf("restored (push failed: %v)", pushErr)}
	}
	return RestoreResult{Name: name, Status: statusRestored}
}
