package repo

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// FailoverConfig holds all dependencies for a failover operation.
type FailoverConfig struct {
	RootDir string
	Runner  GitRunner
	Output  logger.FieldLogger
}

// FailoverResult holds the outcome of failing over a single repository.
type FailoverResult struct {
	Name   string
	Status string
}

// RunFailover switches all repos from GitHub (origin) to Codeberg (origin).
func RunFailover(cfg FailoverConfig) error {
	log := cfg.Output

	repos := FindAllRepos(cfg.RootDir)
	if len(repos) == 0 {
		log.WithField("dir", cfg.RootDir).Warn("no git repositories found")
		return nil
	}

	log.WithField("count", len(repos)).Info("starting failover")

	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([]FailoverResult, len(repos))
	var wg sync.WaitGroup

	for i, repoPath := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			result := failoverSingleRepo(path, cfg.RootDir, cfg.Runner)
			log.WithFields(logger.Fields{
				logFieldRepo:   result.Name,
				logFieldStatus: result.Status,
			}).Info(result.Status)
			results[idx] = result
		}(i, repoPath)
	}

	wg.Wait()

	failoverStatusCategory := map[string]string{
		statusSwitched:                  statusSwitched,
		"skipped (no codeberg remote)":  statusSkipped,
		"skipped (already failed over)": statusSkipped,
	}

	counts := map[string]int{statusSwitched: 0, statusSkipped: 0, mirrorStatusFailed: 0}
	for _, r := range results {
		category, known := failoverStatusCategory[r.Status]
		if !known {
			category = mirrorStatusFailed
		}
		counts[category]++
	}

	log.WithFields(logger.Fields{
		statusSwitched:     counts[statusSwitched],
		statusSkipped:      counts[statusSkipped],
		mirrorStatusFailed: counts[mirrorStatusFailed],
	}).Info("failover completed")
	return nil
}

func failoverSingleRepo(repoPath, rootDir string, runner GitRunner) FailoverResult {
	name, _ := filepath.Rel(rootDir, repoPath)

	// check if already failed over (github remote exists)
	githubURL := runner.Output(repoPath, "remote", "get-url", ProviderGitHub)
	if githubURL != "" {
		return FailoverResult{Name: name, Status: "skipped (already failed over)"}
	}

	// check if codeberg remote exists
	codebergURL := runner.Output(repoPath, "remote", "get-url", ProviderCodeberg)
	if codebergURL == "" {
		return FailoverResult{Name: name, Status: "skipped (no codeberg remote)"}
	}

	// rename origin -> github
	if err := runner.Run(repoPath, "remote", "rename", "origin", ProviderGitHub); err != nil {
		return FailoverResult{Name: name, Status: fmt.Sprintf("FAIL (rename origin: %v)", err)}
	}

	// rename codeberg -> origin
	if err := runner.Run(repoPath, "remote", "rename", ProviderCodeberg, "origin"); err != nil {
		// rollback
		_ = runner.Run(repoPath, "remote", "rename", ProviderGitHub, "origin")
		return FailoverResult{Name: name, Status: fmt.Sprintf("FAIL (rename codeberg: %v)", err)}
	}

	return FailoverResult{Name: name, Status: statusSwitched}
}
