package repo

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	logger "github.com/sirupsen/logrus"
)

const (
	mirrorAPIDelay   = 1 * time.Second
	mirrorMaxWorkers = 4
)

// MirrorConfig holds all dependencies for a mirror operation.
type MirrorConfig struct {
	SourceDir      string
	SSHAlias       string
	DryRun         bool
	SourceProvider globalEntities.ForgeProvider
	TargetProvider globalEntities.ForgeProvider
	Runner         GitRunner
	Output         logger.FieldLogger
}

// MirrorResult holds the outcome of mirroring a single repository.
type MirrorResult struct {
	Name   string
	Status string
}

// RunMirror creates pull mirrors on the target provider for all repos found under SourceDir.
func RunMirror(cfg MirrorConfig) error {
	log := cfg.Output

	providerName, owner, err := DetectProviderAndOwner(cfg.SourceDir)
	if err != nil {
		return err
	}

	if providerName != "github" {
		return fmt.Errorf("mirror only supports GitHub as source provider, detected %q", providerName)
	}

	repos := FindAllRepos(cfg.SourceDir)
	if len(repos) == 0 {
		log.WithField("dir", cfg.SourceDir).Warn("no git repositories found")
		return nil
	}

	log.WithFields(logger.Fields{
		"count": len(repos),
		"owner": owner,
	}).Info("starting mirror")

	mirrorProvider, ok := cfg.TargetProvider.(globalEntities.MirrorProvider)
	if !ok {
		return fmt.Errorf("target provider %q does not support mirroring", cfg.TargetProvider.Name())
	}

	if cfg.DryRun {
		for _, repoPath := range repos {
			name := extractRepoName(repoPath, cfg.SourceDir)
			log.WithField("repo", name).Info("would create mirror")
		}
		return nil
	}

	workers := runtime.NumCPU()
	if workers > mirrorMaxWorkers {
		workers = mirrorMaxWorkers
	}

	sem := make(chan struct{}, workers)
	results := make([]MirrorResult, len(repos))
	var wg sync.WaitGroup

	for i, repoPath := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			time.Sleep(time.Duration(idx) * mirrorAPIDelay / time.Duration(workers))
			results[idx] = mirrorSingleRepo(path, cfg, providerName, owner, mirrorProvider)
		}(i, repoPath)
	}

	wg.Wait()

	mirrorStatusCategory := map[string]string{
		"mirrored":                  "mirrored",
		"mirrored (remote add failed)": "mirrored",
		"skipped (remote exists)":   "skipped",
	}

	mirrored, skipped, failed := 0, 0, 0
	for _, r := range results {
		log.WithFields(logger.Fields{
			"repo":   r.Name,
			"status": r.Status,
		}).Info(r.Status)
		category, ok := mirrorStatusCategory[r.Status]
		if !ok {
			failed++
		} else if category == "mirrored" {
			mirrored++
		} else {
			skipped++
		}
	}

	log.WithFields(logger.Fields{
		"mirrored": mirrored,
		"skipped":  skipped,
		"failed":   failed,
	}).Info("mirror completed")
	return nil
}

func mirrorSingleRepo(
	repoPath string,
	cfg MirrorConfig,
	providerName, owner string,
	mirrorProvider globalEntities.MirrorProvider,
) MirrorResult {
	name := extractRepoName(repoPath, cfg.SourceDir)

	// skip if codeberg remote already exists
	existing := cfg.Runner.Output(repoPath, "remote", "get-url", "codeberg")
	if existing != "" {
		return MirrorResult{Name: name, Status: "skipped (remote exists)"}
	}

	// create mirror on target provider
	sourceHost := ProviderHost(providerName)
	cloneAddr := fmt.Sprintf("https://%s/%s/%s.git", sourceHost, owner, name)
	input := globalEntities.MirrorInput{
		CloneAddr: cloneAddr,
		RepoName:  name,
		RepoOwner: owner,
		Mirror:    true,
		Service:   providerName,
	}

	if migrateErr := mirrorProvider.MigrateRepository(context.Background(), input); migrateErr != nil {
		return MirrorResult{Name: name, Status: fmt.Sprintf("FAIL (%v)", migrateErr)}
	}

	// add codeberg remote locally
	targetHost := ProviderHost("codeberg")
	if cfg.SSHAlias != "" {
		targetHost = fmt.Sprintf("%s-%s", targetHost, cfg.SSHAlias)
	}
	remoteURL := fmt.Sprintf("git@%s:%s/%s.git", targetHost, owner, name)
	if addErr := cfg.Runner.Run(repoPath, "remote", "add", "codeberg", remoteURL); addErr != nil {
		return MirrorResult{Name: name, Status: "mirrored (remote add failed)"}
	}

	return MirrorResult{Name: name, Status: "mirrored"}
}

func extractRepoName(repoPath, sourceDir string) string {
	name, _ := filepath.Rel(sourceDir, repoPath)
	return name
}
