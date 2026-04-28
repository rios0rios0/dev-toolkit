package gist

import (
	"io"
	"runtime"
	"strings"
	"sync"

	"github.com/rios0rios0/devforge/internal/repo"
	logger "github.com/sirupsen/logrus"
)

// RunSync syncs all gists under rootDir in parallel, delegating per-gist work
// to repo.SyncSingleRepo so it benefits from the same fetch/rebase/WIP logic.
func RunSync(rootDir string, runner repo.GitRunner, output io.Writer) error {
	log := repo.NewLogger(output)

	gists := repo.FindAllRepos(rootDir)
	total := len(gists)
	if total == 0 {
		log.WithField("dir", rootDir).Warn("no gist repositories found")
		return nil
	}

	workers := runtime.NumCPU()
	log.WithField("count", total).Info("found gists to sync")

	sem := make(chan struct{}, workers)
	results := make([]repo.SyncResult, total)
	var wg sync.WaitGroup

	for i, path := range gists {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, p string) {
			defer wg.Done()
			defer func() { <-sem }()
			result := repo.SyncSingleRepo(p, rootDir, runner)
			log.WithFields(logger.Fields{
				"gist":   result.Name,
				"status": result.Status,
			}).Info("sync result")
			results[idx] = result
		}(i, path)
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
		"synced": synced,
		"wip":    wip,
		"failed": failed,
	}).Info("summary")
	return nil
}
