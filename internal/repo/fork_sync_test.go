package repo_test

import (
	"context"
	"errors"
	"testing"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/repo"
	"github.com/rios0rios0/devforge/internal/testutil/builders"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
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
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/upstream/HEAD"},
				"refs/remotes/upstream/main",
			).
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
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/upstream/HEAD"},
				"refs/remotes/upstream/main",
			).
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
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/upstream/HEAD"},
				"refs/remotes/upstream/main",
			).
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
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/upstream/HEAD"},
				"refs/remotes/upstream/main",
			).
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
	})
}

// discardWriter is an io.Writer that discards all output.
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func toRepoSlice(repos ...globalEntities.Repository) []globalEntities.Repository {
	return repos
}
