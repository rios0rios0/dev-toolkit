package repo_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestListMergedBranches(t *testing.T) {
	t.Parallel()

	t.Run("should return merged branches excluding default branch", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n* main\n  fix/done")

		// when
		branches := repo.ListMergedBranches("/repo", "main", runner)

		// then
		assert.Equal(t, []string{"feat/old", "fix/done"}, branches)
	})

	t.Run("should return nil when no merged branches", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--merged", "main"}, "* main")

		// when
		branches := repo.ListMergedBranches("/repo", "main", runner)

		// then
		assert.Empty(t, branches)
	})

	t.Run("should return nil when output is empty", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		branches := repo.ListMergedBranches("/repo", "main", runner)

		// then
		assert.Nil(t, branches)
	})

	t.Run("should exclude HEAD pointer entries", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n  HEAD -> main")

		// when
		branches := repo.ListMergedBranches("/repo", "main", runner)

		// then
		assert.Equal(t, []string{"feat/old"}, branches)
	})
}

func TestPruneSingleRepo(t *testing.T) {
	t.Parallel()

	t.Run("should return clean when no merged branches exist", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "* main")

		// when
		result := repo.PruneSingleRepo("/root/my-repo", "/root", runner, false)

		// then
		assert.Equal(t, "my-repo", result.Name)
		assert.Equal(t, "clean", result.Status)
		assert.Empty(t, result.Deleted)
	})

	t.Run("should delete merged branches", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n* main\n  fix/done")

		// when
		result := repo.PruneSingleRepo("/root/my-repo", "/root", runner, false)

		// then
		assert.Equal(t, "my-repo", result.Name)
		assert.Contains(t, result.Status, "deleted 2 branches")
		assert.Equal(t, []string{"feat/old", "fix/done"}, result.Deleted)
	})

	t.Run("should report would-delete in dry-run mode", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n* main")

		// when
		result := repo.PruneSingleRepo("/root/my-repo", "/root", runner, true)

		// then
		assert.Contains(t, result.Status, "would delete")
		assert.Equal(t, []string{"feat/old"}, result.Deleted)
	})

	t.Run("should report partial failure when branch deletion fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n  fix/done\n* main").
			WithRunError([]string{"branch", "-d", "fix/done"}, errors.New("not fully merged"))

		// when
		result := repo.PruneSingleRepo("/root/my-repo", "/root", runner, false)

		// then
		assert.Contains(t, result.Status, "deleted 1")
		assert.Contains(t, result.Status, "failed 1")
		assert.Equal(t, []string{"feat/old"}, result.Deleted)
	})
}

func TestRunPrune(t *testing.T) {
	t.Parallel()

	t.Run("should report no repos when directory is empty", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunPrune(root, runner, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no git repositories found")
	})

	t.Run("should prune repos and report summary", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		createGitRepo(t, root+"/repo-b")
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n* main")
		var buf bytes.Buffer

		// when
		err := repo.RunPrune(root, runner, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "summary")
		assert.Contains(t, buf.String(), "pruned")
	})

	t.Run("should report dry-run mode", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n* main")
		var buf bytes.Buffer

		// when
		err := repo.RunPrune(root, runner, true, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "dry-run")
		assert.Contains(t, buf.String(), "would delete")
	})

	t.Run("should report all clean when no merged branches", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "* main")
		var buf bytes.Buffer

		// when
		err := repo.RunPrune(root, runner, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "clean=1")
	})

	t.Run("should discover and prune nested repositories recursively", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/project-a/repo-one")
		createGitRepo(t, root+"/project-b/repo-two")
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
			WithOutput([]string{"branch", "--merged", "main"}, "  feat/old\n* main")
		var buf bytes.Buffer

		// when
		err := repo.RunPrune(root, runner, false, &buf)

		// then
		require.NoError(t, err)
		output := buf.String()
		assert.NotContains(t, output, "no git repositories found")
		assert.Contains(t, output, "count=2")
		assert.Contains(t, output, "pruned=2")
	})
}
