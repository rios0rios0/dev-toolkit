package repo

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	logger "github.com/sirupsen/logrus"
)

const forkSyncBranch = "fork-sync/upstream"

// ForkSyncConfig holds all dependencies for a fork-sync operation.
type ForkSyncConfig struct {
	RootDir  string
	DryRun   bool
	Provider globalEntities.ForgeProvider
	Resolver ForkResolver
	Runner   GitRunner
	Output   logger.FieldLogger
}

// ForkSyncResult holds the outcome of syncing a single forked repository.
type ForkSyncResult struct {
	Name   string
	Status string
}

// RunForkSync discovers forked repositories and syncs each with its upstream parent.
func RunForkSync(cfg ForkSyncConfig) error {
	log := cfg.Output

	providerName, owner, err := DetectProviderAndOwner(cfg.RootDir)
	if err != nil {
		return err
	}

	log.WithFields(logger.Fields{
		"provider":    providerName,
		logFieldOwner: owner,
	}).Info("fork-sync workflow started")

	remoteRepos, discoverErr := DiscoverRepos(cfg.Provider, owner, false, log)
	if discoverErr != nil {
		return discoverErr
	}

	forks := filterForks(remoteRepos)
	if len(forks) == 0 {
		log.Info("no forked repositories found on remote")
		return nil
	}

	log.WithField("count", len(forks)).Info("forked repositories detected")

	depth := ProviderScanDepth(providerName)
	localRepos := ScanLocalRepos(cfg.RootDir, depth)
	localSet := make(map[string]struct{}, len(localRepos))
	for _, name := range localRepos {
		localSet[name] = struct{}{}
	}

	var localForks []globalEntities.Repository
	for _, f := range forks {
		if _, ok := localSet[Key(f)]; ok {
			localForks = append(localForks, f)
		}
	}

	if len(localForks) == 0 {
		log.Info("no forked repositories found locally")
		return nil
	}

	log.WithField("count", len(localForks)).Info("local forked repositories to sync")

	if cfg.DryRun {
		for _, f := range localForks {
			log.WithField(logFieldRepo, Key(f)).Info("would sync fork with upstream")
		}
		return nil
	}

	results := parallelForkSync(localForks, cfg)
	logForkSyncSummary(results, log)

	return nil
}

func filterForks(repos []globalEntities.Repository) []globalEntities.Repository {
	var forks []globalEntities.Repository
	for _, r := range repos {
		if r.IsFork {
			forks = append(forks, r)
		}
	}
	return forks
}

func parallelForkSync(forks []globalEntities.Repository, cfg ForkSyncConfig) []ForkSyncResult {
	workers := runtime.NumCPU()
	sem := make(chan struct{}, workers)
	results := make([]ForkSyncResult, len(forks))
	var wg sync.WaitGroup

	for i, f := range forks {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, fork globalEntities.Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			repoPath := filepath.Join(cfg.RootDir, Key(fork))
			result := ForkSyncSingleRepo(repoPath, fork, cfg)
			cfg.Output.WithFields(logger.Fields{
				logFieldRepo:   result.Name,
				logFieldStatus: result.Status,
			}).Info("fork-sync result")
			results[idx] = result
		}(i, f)
	}

	wg.Wait()
	return results
}

func ForkSyncSingleRepo(
	repoPath string,
	remoteRepo globalEntities.Repository,
	cfg ForkSyncConfig,
) ForkSyncResult {
	name := Key(remoteRepo)
	defaultBranch := DetectDefaultBranch(repoPath, cfg.Runner)
	currentBranch := DetectCurrentBranch(repoPath, defaultBranch, cfg.Runner)
	isDirty := cfg.Runner.Output(repoPath, "status", "--porcelain") != ""
	wipBranch := fmt.Sprintf("wip/%s", currentBranch)

	// resolve upstream remote and default branch
	upstreamDefault, resolveErr := ensureUpstreamRemote(repoPath, remoteRepo, cfg)
	if resolveErr != nil {
		return ForkSyncResult{Name: name, Status: fmt.Sprintf("FAIL (upstream: %v)", resolveErr)}
	}

	// save WIP state if dirty
	if isDirty {
		if result, ok := SaveWIPState(repoPath, name, currentBranch, wipBranch, cfg.Runner); !ok {
			return ForkSyncResult(result)
		}
	}

	result := syncWithUpstream(repoPath, name, upstreamDefault, currentBranch, wipBranch, isDirty, cfg.Runner)
	return ForkSyncResult{Name: name, Status: result.Status}
}

func ensureUpstreamRemote(
	repoPath string,
	remoteRepo globalEntities.Repository,
	cfg ForkSyncConfig,
) (string, error) {
	existingURL := cfg.Runner.Output(repoPath, "remote", "get-url", "upstream")
	if existingURL != "" {
		// upstream already configured, detect its default branch
		branch := detectUpstreamDefaultBranch(repoPath, cfg.Runner)
		return branch, nil
	}

	// need to add upstream remote via API
	parentInfo, err := cfg.Resolver.GetParentInfo(
		context.Background(), remoteRepo.Organization, remoteRepo.Name,
	)
	if err != nil {
		return "", fmt.Errorf("could not resolve parent: %w", err)
	}

	if addErr := cfg.Runner.Run(repoPath, "remote", "add", "upstream", parentInfo.SSHURL); addErr != nil {
		return "", fmt.Errorf("could not add upstream remote: %w", addErr)
	}

	return parentInfo.DefaultBranch, nil
}

func detectUpstreamDefaultBranch(repoPath string, runner GitRunner) string {
	ref := runner.Output(repoPath, "symbolic-ref", "refs/remotes/upstream/HEAD")
	branch := strings.TrimPrefix(ref, "refs/remotes/upstream/")
	if branch != "" && branch != ref {
		return branch
	}

	// try auto-detecting
	_ = runner.Run(repoPath, "remote", "set-head", "upstream", "--auto")
	ref = runner.Output(repoPath, "symbolic-ref", "refs/remotes/upstream/HEAD")
	branch = strings.TrimPrefix(ref, "refs/remotes/upstream/")
	if branch != "" && branch != ref {
		return branch
	}

	// fallback: use origin's default branch
	return DetectDefaultBranch(repoPath, runner)
}

func syncWithUpstream(
	repoPath, name, upstreamDefault, currentBranch, wipBranch string,
	isDirty bool, runner GitRunner,
) ForkSyncResult {
	// fetch upstream
	if err := runner.Run(repoPath, "fetch", "upstream"); err != nil {
		restoreForkBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
		return ForkSyncResult{Name: name, Status: fmt.Sprintf("FAIL (fetch upstream: %v)", err)}
	}

	// ensure local branch exists for upstream default
	localBranchExists := runner.Output(
		repoPath, "rev-parse", "--verify", "refs/heads/"+upstreamDefault,
	) != ""
	if !localBranchExists {
		if err := runner.Run(
			repoPath, "checkout", "-b", upstreamDefault,
			"upstream/"+upstreamDefault,
		); err != nil {
			restoreForkBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
			return ForkSyncResult{
				Name:   name,
				Status: fmt.Sprintf("FAIL (create branch %s: %v)", upstreamDefault, err),
			}
		}
	} else {
		if err := runner.Run(repoPath, "checkout", upstreamDefault); err != nil {
			restoreForkBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
			return ForkSyncResult{
				Name:   name,
				Status: fmt.Sprintf("FAIL (checkout %s: %v)", upstreamDefault, err),
			}
		}
	}

	// rebase onto upstream
	if err := runner.Run(repoPath, "rebase", "upstream/"+upstreamDefault); err != nil {
		return handleRebaseConflict(repoPath, name, upstreamDefault, currentBranch, wipBranch, isDirty, runner)
	}

	// push updated branch to origin
	if err := runner.Run(repoPath, "push", "origin", upstreamDefault); err != nil {
		// try force push in case of divergence
		if forceErr := runner.Run(repoPath, "push", "origin", upstreamDefault, "--force-with-lease"); forceErr != nil {
			restoreForkBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
			return ForkSyncResult{
				Name:   name,
				Status: fmt.Sprintf("FAIL (push origin: %v)", forceErr),
			}
		}
	}

	return restoreAfterForkSync(repoPath, name, upstreamDefault, currentBranch, wipBranch, isDirty, runner)
}

func handleRebaseConflict(
	repoPath, name, upstreamDefault, currentBranch, wipBranch string,
	isDirty bool, runner GitRunner,
) ForkSyncResult {
	_ = runner.Run(repoPath, "rebase", "--abort")

	// create a reference branch pointing to upstream HEAD
	if err := runner.Run(repoPath, "branch", "-f", forkSyncBranch, "upstream/"+upstreamDefault); err != nil {
		restoreForkBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
		return ForkSyncResult{
			Name:   name,
			Status: fmt.Sprintf("FAIL (create reference branch %s: %v)", forkSyncBranch, err),
		}
	}

	if err := runner.Run(repoPath, "push", "-u", "origin", forkSyncBranch, "--force"); err != nil {
		restoreForkBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
		return ForkSyncResult{
			Name:   name,
			Status: fmt.Sprintf("FAIL (push reference branch %s to origin: %v)", forkSyncBranch, err),
		}
	}

	restoreForkBranch(repoPath, currentBranch, wipBranch, isDirty, runner)

	return ForkSyncResult{
		Name:   name,
		Status: fmt.Sprintf("conflict (%s pushed to origin)", forkSyncBranch),
	}
}

func restoreForkBranch(repoPath, currentBranch, wipBranch string, isDirty bool, runner GitRunner) {
	RestoreBranch(repoPath, currentBranch, wipBranch, isDirty, runner)
}

func restoreAfterForkSync(
	repoPath, name, upstreamDefault, currentBranch, wipBranch string,
	isDirty bool, runner GitRunner,
) ForkSyncResult {
	status := statusSynced
	if isDirty {
		_ = runner.Run(repoPath, "checkout", wipBranch)
		if err := runner.Run(repoPath, "rebase", upstreamDefault); err != nil {
			_ = runner.Run(repoPath, "rebase", "--abort")
		}
		_ = runner.Run(repoPath, "checkout", currentBranch)
		status = fmt.Sprintf("synced (wip: %s)", wipBranch)
	} else {
		_ = runner.Run(repoPath, "checkout", currentBranch)
	}
	return ForkSyncResult{Name: name, Status: status}
}

func logForkSyncSummary(results []ForkSyncResult, log logger.FieldLogger) {
	synced, conflicts, failed := 0, 0, 0
	for _, r := range results {
		switch {
		case strings.HasPrefix(r.Status, "synced"):
			synced++
		case strings.HasPrefix(r.Status, "conflict"):
			conflicts++
		case strings.HasPrefix(r.Status, "FAIL"):
			failed++
		}
	}

	log.WithFields(logger.Fields{
		statusSynced:       synced,
		"conflicts":        conflicts,
		mirrorStatusFailed: failed,
	}).Info("fork-sync summary")
}
