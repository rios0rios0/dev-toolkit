package repo

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
	logger "github.com/sirupsen/logrus"
)

const (
	mirrorAPIDelay     = 1 * time.Second
	mirrorMaxWorkers   = 4
	maxMirrorArgs      = 2
	mirrorStatusFailed = "failed"
)

// MirrorConfig holds all dependencies for a mirror operation. There is no source provider:
// the repositories are found on disk and Codeberg pulls each one from its public address, so
// mirror never calls the source provider's API.
type MirrorConfig struct {
	SourceDir      string
	SSHAlias       string
	DryRun         bool
	TargetProvider globalEntities.ForgeProvider
	// TargetCredentialSource says where TargetProvider's credential came from
	// ("CODEBERG_TOKEN environment variable"), for the log; it never holds the token.
	TargetCredentialSource string
	Runner                 GitRunner
	Output                 logger.FieldLogger
}

// MirrorResult holds the outcome of mirroring a single repository.
type MirrorResult struct {
	Name   string
	Status string
}

// ResolveMirrorTarget builds the Codeberg provider that mirror creates its pull mirrors on, from
// gitforge's default credential chain; see ResolveMirrorTargetWith.
func ResolveMirrorTarget() (globalEntities.ForgeProvider, string, error) {
	return ResolveMirrorTargetWith(newCredentialResolver())
}

// ResolveMirrorTargetWith builds the Codeberg provider that mirror creates its pull mirrors on,
// from the credential the given resolver supplies, and returns where that credential came
// from. It is the only credential mirror needs: Codeberg pulls each repository from its public
// address on the source provider, so no source provider credential is asked for -- for
// GitHub, one that would run gh for nothing.
func ResolveMirrorTargetWith(
	resolver credentialEntities.CredentialResolver,
) (globalEntities.ForgeProvider, string, error) {
	return ResolveProviderWith(ProviderCodeberg, "", resolver)
}

// RunMirror creates pull mirrors on the target provider for all repos found under SourceDir.
func RunMirror(cfg MirrorConfig) error {
	log := cfg.Output

	providerName, owner, err := DetectProviderAndOwner(cfg.SourceDir)
	if err != nil {
		return err
	}

	if providerName != ProviderGitHub {
		return fmt.Errorf("mirror only supports GitHub as source provider, detected %q", providerName)
	}

	repos := FindAllRepos(cfg.SourceDir)
	if len(repos) == 0 {
		log.WithField("dir", cfg.SourceDir).Warn("no git repositories found")
		return nil
	}

	log.WithFields(logger.Fields{
		"count":            len(repos),
		logFieldOwner:      owner,
		logFieldCredential: cfg.TargetCredentialSource,
	}).Info("starting mirror")

	mirrorProvider, ok := cfg.TargetProvider.(globalEntities.MirrorProvider)
	if !ok {
		return fmt.Errorf("target provider %q does not support mirroring", cfg.TargetProvider.Name())
	}

	if cfg.DryRun {
		for _, repoPath := range repos {
			name := extractRepoName(repoPath, cfg.SourceDir)
			log.WithField(logFieldRepo, name).Info("would create mirror")
		}
		return nil
	}

	workers := min(runtime.NumCPU(), mirrorMaxWorkers)

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
			result := mirrorSingleRepo(path, cfg, providerName, owner, mirrorProvider)
			log.WithFields(logger.Fields{
				logFieldRepo:   result.Name,
				logFieldStatus: result.Status,
			}).Info(result.Status)
			results[idx] = result
		}(i, repoPath)
	}

	wg.Wait()

	mirrorStatusCategory := map[string]string{
		statusMirrored:                 statusMirrored,
		"mirrored (remote add failed)": statusMirrored,
		"skipped (remote exists)":      statusSkipped,
	}

	counts := map[string]int{statusMirrored: 0, statusSkipped: 0, mirrorStatusFailed: 0}
	for _, r := range results {
		category, known := mirrorStatusCategory[r.Status]
		if !known {
			category = mirrorStatusFailed
		}
		counts[category]++
	}

	log.WithFields(logger.Fields{
		statusMirrored:     counts[statusMirrored],
		statusSkipped:      counts[statusSkipped],
		mirrorStatusFailed: counts[mirrorStatusFailed],
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

	return MirrorResult{Name: name, Status: statusMirrored}
}

func extractRepoName(repoPath, sourceDir string) string {
	name, _ := filepath.Rel(sourceDir, repoPath)
	return name
}

// MaxMirrorArgs returns the maximum number of positional arguments for the mirror command.
func MaxMirrorArgs() int {
	return maxMirrorArgs
}
