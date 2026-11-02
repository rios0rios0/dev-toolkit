package repo_test

import (
	"context"
	"errors"
	"testing"

	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/builders"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestRunForkSync(t *testing.T) {
	t.Parallel()

	t.Run("should report no forks when none exist on remote", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir() + "/github.com/owner"
		createGitRepo(t, root+"/repo-a")
		provider := doubles.NewForgeProviderStub().WithRepos(nil)
		resolver := doubles.NewForkResolverStub()
		runner := doubles.NewGitRunnerStub()
		log := repo.NewLogger(&discardWriter{})

		// when
		err := repo.RunForkSync(repo.ForkSyncConfig{
			RootDir:  root,
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		})

		// then
		require.NoError(t, err)
	})

	t.Run("should skip non-fork repositories", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir() + "/github.com/owner"
		createGitRepo(t, root+"/repo-a")
		nonFork := builders.NewRepositoryBuilder().
			WithName("repo-a").
			WithOrganization("owner").
			WithFork(false).
			Build()
		provider := doubles.NewForgeProviderStub().WithRepos(toRepoSlice(nonFork))
		resolver := doubles.NewForkResolverStub()
		runner := doubles.NewGitRunnerStub()
		log := repo.NewLogger(&discardWriter{})

		// when
		err := repo.RunForkSync(repo.ForkSyncConfig{
			RootDir:  root,
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		})

		// then
		require.NoError(t, err)
	})

	t.Run("should sync fork and report summary", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir() + "/github.com/owner"
		createGitRepo(t, root+"/forked-repo")
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		provider := doubles.NewForgeProviderStub().WithRepos(toRepoSlice(fork))
		resolver := doubles.NewForkResolverStub().WithParentInfo(&repo.ParentInfo{
			SSHURL:        "git@github.com:upstream-org/forked-repo.git",
			DefaultBranch: "main",
		})
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "").
			WithOutput([]string{"rev-parse", "--verify", "refs/heads/main"}, "abc123")
		log := repo.NewLogger(&discardWriter{})

		// when
		err := repo.RunForkSync(repo.ForkSyncConfig{
			RootDir:  root,
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		})

		// then
		require.NoError(t, err)
	})

	t.Run("should log forks in dry-run mode without syncing", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir() + "/github.com/owner"
		createGitRepo(t, root+"/forked-repo")
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		provider := doubles.NewForgeProviderStub().WithRepos(toRepoSlice(fork))
		resolver := doubles.NewForkResolverStub()
		runner := doubles.NewGitRunnerStub()
		log := repo.NewLogger(&discardWriter{})

		// when
		err := repo.RunForkSync(repo.ForkSyncConfig{
			RootDir:  root,
			DryRun:   true,
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		})

		// then
		require.NoError(t, err)
	})

	t.Run("should report error when provider discovery fails", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir() + "/github.com/owner"
		provider := doubles.NewForgeProviderStub().
			WithDiscoverError(errors.New("API error"))
		resolver := doubles.NewForkResolverStub()
		runner := doubles.NewGitRunnerStub()
		log := repo.NewLogger(&discardWriter{})

		// when
		err := repo.RunForkSync(repo.ForkSyncConfig{
			RootDir:  root,
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})
}

func TestForkSyncSingleRepo(t *testing.T) {
	t.Parallel()

	t.Run("should add upstream remote when not present and sync", func(t *testing.T) {
		t.Parallel()
		// given
		var addedRemoteURL string
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "").
			WithOutput([]string{"remote", "get-url", "upstream"}, "").
			WithOutput([]string{"rev-parse", "--verify", "refs/heads/main"}, "abc123")
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) >= 4 && args[0] == "remote" && args[1] == "add" && args[2] == "upstream" {
				addedRemoteURL = args[3]
			}
			return nil
		}
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		resolver := doubles.NewForkResolverStub().WithParentInfo(&repo.ParentInfo{
			SSHURL:        "git@github.com:upstream-org/forked-repo.git",
			DefaultBranch: "main",
		})
		provider := doubles.NewForgeProviderStub()
		log := repo.NewLogger(&discardWriter{})

		cfg := repo.ForkSyncConfig{
			RootDir:  "/root",
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		}

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Equal(t, "forked-repo", result.Name)
		assert.Equal(t, "synced", result.Status)
		assert.Equal(t, "git@github.com:upstream-org/forked-repo.git", addedRemoteURL)
	})

	t.Run("should reuse existing upstream remote without API call", func(t *testing.T) {
		t.Parallel()
		// given
		apiCalled := false
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "").
			WithOutput([]string{"remote", "get-url", "upstream"}, "git@github.com:existing/repo.git").
			WithRemoteHEAD("upstream", "main").
			WithOutput([]string{"rev-parse", "--verify", "refs/heads/main"}, "abc123")
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		resolver := &doubles.ForkResolverStub{
			GetParentInfoFunc: func(_ context.Context, _, _ string) (*repo.ParentInfo, error) {
				apiCalled = true
				return &repo.ParentInfo{SSHURL: "unused", DefaultBranch: "main"}, nil
			},
		}
		provider := doubles.NewForgeProviderStub()
		log := repo.NewLogger(&discardWriter{})

		cfg := repo.ForkSyncConfig{
			RootDir:  "/root",
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		}

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.False(t, apiCalled, "should not call API when upstream remote exists")
	})

	t.Run("should preserve WIP state for dirty repo", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "feat/x").
			WithOutput([]string{"status", "--porcelain"}, " M file.go").
			WithOutput([]string{"remote", "get-url", "upstream"}, "git@github.com:upstream/repo.git").
			WithRemoteHEAD("upstream", "main").
			WithOutput([]string{"rev-parse", "--verify", "refs/heads/main"}, "abc123")
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		resolver := doubles.NewForkResolverStub()
		provider := doubles.NewForgeProviderStub()
		log := repo.NewLogger(&discardWriter{})

		cfg := repo.ForkSyncConfig{
			RootDir:  "/root",
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		}

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Contains(t, result.Status, "synced (wip:")
	})

	t.Run("should create reference branch on rebase conflict", func(t *testing.T) {
		t.Parallel()
		// given
		var pushedBranch string
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "").
			WithOutput([]string{"remote", "get-url", "upstream"}, "git@github.com:upstream/repo.git").
			WithRemoteHEAD("upstream", "main").
			WithOutput([]string{"rev-parse", "--verify", "refs/heads/main"}, "abc123").
			WithRunError([]string{"rebase", "upstream/main"}, errors.New("conflict"))
		origRun := runner.RunFunc
		runner.RunFunc = func(dir string, args ...string) error {
			if len(args) >= 4 && args[0] == "push" && args[1] == "-u" {
				pushedBranch = args[3]
			}
			return origRun(dir, args...)
		}
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		resolver := doubles.NewForkResolverStub()
		provider := doubles.NewForgeProviderStub()
		log := repo.NewLogger(&discardWriter{})

		cfg := repo.ForkSyncConfig{
			RootDir:  "/root",
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		}

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Contains(t, result.Status, "conflict")
		assert.Contains(t, result.Status, "fork-sync/upstream")
		assert.Equal(t, "fork-sync/upstream", pushedBranch)
	})

	t.Run("should report failure when fetch upstream fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "").
			WithOutput([]string{"remote", "get-url", "upstream"}, "git@github.com:upstream/repo.git").
			WithRemoteHEAD("upstream", "main").
			WithOutput([]string{"rev-parse", "--verify", "refs/heads/main"}, "abc123").
			WithRunError([]string{"fetch", "upstream"}, errors.New("network error"))
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		resolver := doubles.NewForkResolverStub()
		provider := doubles.NewForgeProviderStub()
		log := repo.NewLogger(&discardWriter{})

		cfg := repo.ForkSyncConfig{
			RootDir:  "/root",
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		}

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Contains(t, result.Status, "FAIL")
		assert.Contains(t, result.Status, "fetch upstream")
	})

	t.Run("should report failure when GetParentInfo fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "").
			WithOutput([]string{"remote", "get-url", "upstream"}, "")
		fork := builders.NewRepositoryBuilder().
			WithName("forked-repo").
			WithOrganization("owner").
			WithFork(true).
			Build()
		resolver := doubles.NewForkResolverStub().WithError(errors.New("API error"))
		provider := doubles.NewForgeProviderStub()
		log := repo.NewLogger(&discardWriter{})

		cfg := repo.ForkSyncConfig{
			RootDir:  "/root",
			Provider: provider,
			Resolver: resolver,
			Runner:   runner,
			Output:   log,
		}

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Contains(t, result.Status, "FAIL")
		assert.Contains(t, result.Status, "upstream")
		assert.Contains(t, result.Status, "could not resolve parent")
	})

	t.Run("should refresh upstream/HEAD before detecting the upstream default branch", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithRemoteHEAD("upstream", "main").
			WithRef("refs/heads/main").
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"remote", "get-url", "upstream"}, "git@github.com:upstream/repo.git")
		calls := recordRunCalls(runner)
		fork, cfg := forkSyncCase(runner)

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Equal(t, "synced", result.Status)
		require.NotEmpty(t, *calls)
		assert.Equal(t, "remote set-head upstream --auto", (*calls)[0])
	})

	t.Run("should rebase onto the branch upstream reports when upstream/HEAD names a deleted one", func(t *testing.T) {
		t.Parallel()
		// given: upstream renamed master to main after upstream/HEAD was recorded
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithRef("refs/heads/main").
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"remote", "get-url", "upstream"}, "git@github.com:upstream/repo.git").
			WithOutput([]string{"symbolic-ref", "refs/remotes/upstream/HEAD"}, "refs/remotes/upstream/master").
			WithOutput([]string{"ls-remote", "--symref", "upstream", "HEAD"}, advertisedHEAD("main"))
		calls := recordRunCalls(runner)
		fork, cfg := forkSyncCase(runner)

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Contains(t, *calls, "rebase upstream/main")
		assert.NotContains(t, *calls, "rebase upstream/master")
	})

	t.Run("should fail instead of borrowing origin's default when upstream's is unknown", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithRef("refs/heads/main").
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"remote", "get-url", "upstream"}, "git@github.com:upstream/repo.git")
		calls := recordRunCalls(runner)
		fork, cfg := forkSyncCase(runner)

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Equal(t, "FAIL (upstream: default branch not found for remote upstream)", result.Status)
		assert.Equal(t, []string{"remote set-head upstream --auto"}, *calls)
	})

	t.Run("should fail before touching the fork when origin's default branch is unknown", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "feat/x").
			WithOutput([]string{"status", "--porcelain"}, " M file.go")
		calls := recordRunCalls(runner)
		fork, cfg := forkSyncCase(runner)

		// when
		result := repo.ForkSyncSingleRepo("/root/forked-repo", fork, cfg)

		// then
		assert.Equal(t, "FAIL (default branch not found for remote origin)", result.Status)
		assert.Empty(t, *calls)
	})
}

// forkSyncCase builds the fork and the config a single-repository fork-sync test needs,
// with git answering through runner.
func forkSyncCase(runner *doubles.GitRunnerStub) (globalEntities.Repository, repo.ForkSyncConfig) {
	fork := builders.NewRepositoryBuilder().
		WithName("forked-repo").
		WithOrganization("owner").
		WithFork(true).
		Build()
	return fork, repo.ForkSyncConfig{
		RootDir:  "/root",
		Provider: doubles.NewForgeProviderStub(),
		Resolver: doubles.NewForkResolverStub(),
		Runner:   runner,
		Output:   repo.NewLogger(&discardWriter{}),
	}
}

// discardWriter is an [io.Writer] that discards all output.
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func toRepoSlice(repos ...globalEntities.Repository) []globalEntities.Repository {
	return repos
}
