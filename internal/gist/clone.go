package gist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/rios0rios0/devforge/internal/repo"
	logger "github.com/sirupsen/logrus"
)

const maxCloneArgs = 2

// MaxCloneArgs returns the maximum positional arg count for the clone command.
func MaxCloneArgs() int {
	return maxCloneArgs
}

// PreflightFunc verifies SSH connectivity before cloning.
type PreflightFunc func(sshAlias string, log logger.FieldLogger) error

// CloneConfig wires dependencies for the gist clone workflow.
type CloneConfig struct {
	RootDir   string
	Owner     string
	SSHAlias  string
	DryRun    bool
	Provider  Provider
	Runner    repo.GitRunner
	Output    io.Writer
	Logger    logger.FieldLogger
	Preflight PreflightFunc
}

func (c *CloneConfig) log() logger.FieldLogger {
	if c.Logger == nil {
		c.Logger = repo.NewLogger(c.Output)
	}
	return c.Logger
}

// RunClone discovers gists for the owner and clones the missing ones.
func RunClone(cfg CloneConfig) error {
	log := cfg.log()

	if cfg.Owner == "" {
		return errors.New("owner must be provided to clone gists")
	}

	log.WithField("owner", cfg.Owner).Info("gist clone workflow started")
	if cfg.DryRun {
		log.Info("dry-run mode enabled")
	}

	remote, err := cfg.Provider.ListGists(context.Background(), cfg.Owner)
	if err != nil {
		return fmt.Errorf("failed to discover gists: %w", err)
	}
	log.WithField("count", len(remote)).Info("remote gists discovered")

	local := ScanLocalGists(cfg.RootDir)
	log.WithField("count", len(local)).Info("scanned local gists")

	missing, extra := ComputeDiff(remote, local)
	log.WithFields(logger.Fields{
		"missing": len(missing),
		"extra":   len(extra),
	}).Info("computed gist diff")

	if len(missing) == 0 && len(extra) == 0 {
		log.Info("everything is in sync")
		return nil
	}

	cloned, failed := CloneMissing(missing, cfg)
	for _, name := range extra {
		log.WithField("gist", name).Warn("extra local gist (not on remote)")
	}

	log.WithFields(logger.Fields{
		"cloned": cloned,
		"failed": failed,
		"extra":  len(extra),
	}).Info("gist clone workflow completed")
	return nil
}

// ComputeDiff computes which remote gists are missing locally and which local
// directories are not present on the remote. Keys are assigned via AssignKeys
// so two gists with colliding slugs each get a unique on-disk path.
func ComputeDiff(remote []Gist, local []string) ([]Gist, []string) {
	keys := AssignKeys(remote)

	remoteSet := make(map[string]Gist, len(remote))
	for _, g := range remote {
		remoteSet[keys[g.ID]] = g
	}

	localSet := make(map[string]struct{}, len(local))
	for _, name := range local {
		localSet[name] = struct{}{}
	}

	var missing []Gist
	for key, g := range remoteSet {
		if _, ok := localSet[key]; !ok {
			missing = append(missing, g)
		}
	}

	var extra []string
	for _, name := range local {
		if _, ok := remoteSet[name]; !ok {
			extra = append(extra, name)
		}
	}

	return missing, extra
}

// CloneMissing clones the missing gists, honouring dry-run mode.
func CloneMissing(missing []Gist, cfg CloneConfig) (int, int) {
	log := cfg.log()

	if len(missing) == 0 {
		return 0, 0
	}

	keys := AssignKeys(missing)

	if cfg.DryRun {
		for _, g := range missing {
			url := SSHCloneURL(g, cfg.SSHAlias)
			key := keys[g.ID]
			target := filepath.Join(cfg.RootDir, key)
			log.WithFields(logger.Fields{
				"gist":   key,
				"url":    url,
				"target": target,
			}).Info("would clone gist")
		}
		return 0, 0
	}

	preflight := cfg.Preflight
	if preflight == nil {
		preflight = SSHPreflight
	}
	if preflightErr := preflight(cfg.SSHAlias, log); preflightErr != nil {
		log.WithError(preflightErr).Error("SSH preflight failed")
		return 0, len(missing)
	}

	return parallelCloneWithKeys(missing, keys, cfg.SSHAlias, cfg.RootDir, cfg.Runner, log)
}

// SSHPreflight verifies SSH access to gist.github.com via the SSH config alias.
// It must check gist.github.com (not github.com) because gists are served from
// a separate SSH host that may have its own SSH config alias.
func SSHPreflight(sshAlias string, log logger.FieldLogger) error {
	return repo.SSHPreflightHost(gistHost, sshAlias, log)
}

type cloneResult struct {
	name    string
	success bool
}

// ParallelClone clones the given gists concurrently using a worker pool.
func ParallelClone(
	gists []Gist, sshAlias, rootDir string, runner repo.GitRunner, log logger.FieldLogger,
) (int, int) {
	return parallelCloneWithKeys(gists, AssignKeys(gists), sshAlias, rootDir, runner, log)
}

func parallelCloneWithKeys(
	gists []Gist,
	keys map[string]string,
	sshAlias, rootDir string,
	runner repo.GitRunner,
	log logger.FieldLogger,
) (int, int) {
	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([]cloneResult, len(gists))
	var wg sync.WaitGroup

	log.WithFields(logger.Fields{
		"count":   len(gists),
		"workers": workers,
	}).Info("starting parallel gist clone")

	for i, g := range gists {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, g Gist) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = cloneSingle(g, keys[g.ID], sshAlias, rootDir, runner, log)
		}(i, g)
	}
	wg.Wait()

	cloned, failed := 0, 0
	for _, r := range results {
		if r.success {
			cloned++
		} else {
			failed++
		}
	}
	log.WithFields(logger.Fields{
		"cloned": cloned,
		"failed": failed,
	}).Info("parallel gist clone completed")
	return cloned, failed
}

func cloneSingle(
	g Gist, key, sshAlias, rootDir string, runner repo.GitRunner, log logger.FieldLogger,
) cloneResult {
	url := SSHCloneURL(g, sshAlias)
	target := filepath.Join(rootDir, key)

	log.WithFields(logger.Fields{
		"gist":   key,
		"url":    url,
		"target": target,
	}).Info("cloning gist")

	if err := runner.Clone(url, target); err != nil {
		log.WithField("gist", key).WithError(err).Error("clone failed")
		return cloneResult{name: key}
	}

	log.WithFields(logger.Fields{
		"gist":   key,
		"target": target,
	}).Info("gist cloned")
	return cloneResult{name: key, success: true}
}
