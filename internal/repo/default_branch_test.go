package repo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

// advertisedHEAD is what `git ls-remote --symref <remote> HEAD` prints for a remote whose
// HEAD points at branch.
func advertisedHEAD(branch string) string {
	return "ref: refs/heads/" + branch + "\tHEAD\nabc123\tHEAD"
}

func TestDetectDefaultBranch(t *testing.T) {
	t.Parallel()

	t.Run("should return the branch origin/HEAD points to when that branch exists", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().WithRemoteHEAD("origin", "develop")

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "develop", branch)
	})

	t.Run("should ask the remote when origin/HEAD was never recorded", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"ls-remote", "--symref", "origin", "HEAD"}, advertisedHEAD("master"))

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})

	t.Run("should ask the remote when origin/HEAD points to a branch that no longer exists", func(t *testing.T) {
		t.Parallel()
		// given: the remote renamed master to main and a pruning fetch removed origin/master
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/master").
			WithOutput([]string{"ls-remote", "--symref", "origin", "HEAD"}, advertisedHEAD("main")).
			WithRef("refs/heads/master")

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("should keep slashes in a default branch name", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"ls-remote", "--symref", "origin", "HEAD"}, advertisedHEAD("release/2.x"))

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "release/2.x", branch)
	})

	t.Run("should fall back to a conventional branch of origin when the remote cannot be reached", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().WithRef("refs/remotes/origin/master")

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})

	t.Run("should prefer main when origin has both conventional branches", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRef("refs/remotes/origin/master").
			WithRef("refs/remotes/origin/main")

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("should fall back to a conventional local branch when origin is unusable", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().WithRef("refs/heads/master")

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})

	t.Run("should return ErrDefaultBranchNotFound instead of assuming main", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().WithRef("refs/heads/develop")

		// when
		branch, err := repo.DetectDefaultBranch("/repo", runner)

		// then
		require.ErrorIs(t, err, repo.ErrDefaultBranchNotFound)
		assert.Empty(t, branch)
	})
}

func TestDetectRemoteDefaultBranch(t *testing.T) {
	t.Parallel()

	t.Run("should return the branch the named remote's HEAD points to", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithRemoteHEAD("upstream", "trunk")

		// when
		branch, err := repo.DetectRemoteDefaultBranch("/repo", "upstream", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "trunk", branch)
	})

	t.Run("should ask the named remote when its HEAD was never recorded", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"ls-remote", "--symref", "upstream", "HEAD"}, advertisedHEAD("develop"))

		// when
		branch, err := repo.DetectRemoteDefaultBranch("/repo", "upstream", runner)

		// then
		require.NoError(t, err)
		assert.Equal(t, "develop", branch)
	})

	t.Run("should not fall back to local branches, which say nothing about another remote", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithRef("refs/heads/main")

		// when
		branch, err := repo.DetectRemoteDefaultBranch("/repo", "upstream", runner)

		// then
		require.ErrorIs(t, err, repo.ErrDefaultBranchNotFound)
		assert.Contains(t, err.Error(), "upstream")
		assert.Empty(t, branch)
	})
}
