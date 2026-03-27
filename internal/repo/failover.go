package repo

import (
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
			results[idx] = failoverSingleRepo(path, cfg.RootDir, cfg.Runner)
		}(i, repoPath)
	}

	wg.Wait()

	switched, skipped, failed := 0, 0, 0
	for _, r := range results {
		log.WithFields(logger.Fields{
			"repo":   r.Name,
			"status": r.Status,
		}).Info(r.Status)
		switch r.Status {
		case "switched":
			switched++
		case "skipped (no codeberg remote)", "skipped (already failed over)":
			skipped++
		default:
			failed++
		}
	}

	log.WithFields(logger.Fields{
		"switched": switched,
		"skipped":  skipped,
		"failed":   failed,
	}).Info("failover completed")
	return nil
}

func failoverSingleRepo(repoPath, rootDir string, runner GitRunner) FailoverResult {
	name := repoPath[len(rootDir)+1:]

	// check if already failed over (github remote exists)
	githubURL := runner.Output(repoPath, "remote", "get-url", "github")
	if githubURL != "" {
		return FailoverResult{Name: name, Status: "skipped (already failed over)"}
	}

	// check if codeberg remote exists
	codebergURL := runner.Output(repoPath, "remote", "get-url", "codeberg")
	if codebergURL == "" {
		return FailoverResult{Name: name, Status: "skipped (no codeberg remote)"}
	}

	// rename origin -> github
	if err := runner.Run(repoPath, "remote", "rename", "origin", "github"); err != nil {
		return FailoverResult{Name: name, Status: "FAIL (rename origin)"}
	}

	// rename codeberg -> origin
	if err := runner.Run(repoPath, "remote", "rename", "codeberg", "origin"); err != nil {
		// rollback
		_ = runner.Run(repoPath, "remote", "rename", "github", "origin")
		return FailoverResult{Name: name, Status: "FAIL (rename codeberg)"}
	}

	return FailoverResult{Name: name, Status: "switched"}
}
