package repo_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/repo"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestDetectDefaultBranch(t *testing.T) {
	t.Parallel()

	t.Run("should return branch name from symbolic-ref output", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/develop")

		// when
		branch := repo.DetectDefaultBranch("/repo", runner)

		// then
		assert.Equal(t, "develop", branch)
	})

	t.Run("should return main when symbolic-ref returns empty", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		branch := repo.DetectDefaultBranch("/repo", runner)

		// then
		assert.Equal(t, "main", branch)
	})
}

func TestDetectCurrentBranch(t *testing.T) {
	t.Parallel()

	t.Run("should return current branch name", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "feat/my-feature")

		// when
		branch := repo.DetectCurrentBranch("/repo", "main", runner)

		// then
		assert.Equal(t, "feat/my-feature", branch)
	})

	t.Run("should return default branch when detached HEAD", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		branch := repo.DetectCurrentBranch("/repo", "main", runner)

		// then
		assert.Equal(t, "main", branch)
	})
}

func TestSaveWIPState(t *testing.T) {
	t.Parallel()

	t.Run("should return ok when WIP state is saved successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		_, ok := repo.SaveWIPState("/repo", "my-repo", "main", "wip/main", runner)

		// then
		assert.True(t, ok)
	})

	t.Run("should return failure when checkout -B fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"checkout", "-B", "wip/main"}, errors.New("checkout failed"))

		// when
		result, ok := repo.SaveWIPState("/repo", "my-repo", "main", "wip/main", runner)

		// then
		assert.False(t, ok)
		assert.Contains(t, result.Status, "FAIL (wip branch")
	})

	t.Run("should cleanup and fail when git add fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"add", "-A"}, errors.New("add failed"))

		// when
		result, ok := repo.SaveWIPState("/repo", "my-repo", "main", "wip/main", runner)

		// then
		assert.False(t, ok)
		assert.Contains(t, result.Status, "FAIL (wip add")
	})

	t.Run("should cleanup and fail when commit fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) > 0 && args[0] == "commit" {
				return errors.New("commit failed")
			}
			return nil
		}

		// when
		result, ok := repo.SaveWIPState("/repo", "my-repo", "main", "wip/main", runner)

		// then
		assert.False(t, ok)
		assert.Contains(t, result.Status, "FAIL (wip commit")
	})
}

func TestSyncAndRestore(t *testing.T) {
	t.Parallel()

	t.Run("should return synced status for clean repo", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "main", "main", "wip/main", false, runner)

		// then
		assert.Equal(t, "synced", result.Status)
	})

	t.Run("should return synced with wip status for dirty repo", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "main", "feat/x", "wip/feat/x", true, runner)

		// then
		assert.Contains(t, result.Status, "synced (wip:")
	})

	t.Run("should return failure when checkout default branch fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"checkout", "main"}, errors.New("checkout failed"))

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "main", "feat/x", "wip/feat/x", false, runner)

		// then
		assert.Contains(t, result.Status, "FAIL (checkout main")
	})

	t.Run("should return failure when fetch fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"fetch", "--all", "--prune", "-q"}, errors.New("network error"))

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "main", "feat/x", "wip/feat/x", false, runner)

		// then
		assert.Contains(t, result.Status, "FAIL (fetch")
	})

	t.Run("should return failure when pull rebase fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"pull", "--rebase"}, errors.New("conflict"))

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "main", "feat/x", "wip/feat/x", false, runner)

		// then
		assert.Contains(t, result.Status, "FAIL (pull --rebase")
	})
}

func TestRestoreBranch(t *testing.T) {
	t.Parallel()

	t.Run("should checkout wip branch when dirty", func(t *testing.T) {
		t.Parallel()
		// given
		var checkedOut string
		runner := doubles.NewGitRunnerStub()
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) > 0 && args[0] == "checkout" {
				checkedOut = args[1]
			}
			return nil
		}

		// when
		repo.RestoreBranch("/repo", "feat/x", "wip/feat/x", true, runner)

		// then
		assert.Equal(t, "wip/feat/x", checkedOut)
	})

	t.Run("should checkout current branch when clean", func(t *testing.T) {
		t.Parallel()
		// given
		var checkedOut string
		runner := doubles.NewGitRunnerStub()
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) > 0 && args[0] == "checkout" {
				checkedOut = args[1]
			}
			return nil
		}

		// when
		repo.RestoreBranch("/repo", "feat/x", "wip/feat/x", false, runner)

		// then
		assert.Equal(t, "feat/x", checkedOut)
	})
}

func TestRestoreAfterSync(t *testing.T) {
	t.Parallel()

	t.Run("should return synced for clean repo on default branch", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		result := repo.RestoreAfterSync("/repo", "my-repo", "main", "main", "wip/main", false, runner)

		// then
		assert.Equal(t, "synced", result.Status)
	})

	t.Run("should checkout original branch when not on default", func(t *testing.T) {
		t.Parallel()
		// given
		var checkedOut string
		runner := doubles.NewGitRunnerStub()
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) > 0 && args[0] == "checkout" {
				checkedOut = args[1]
			}
			return nil
		}

		// when
		result := repo.RestoreAfterSync("/repo", "my-repo", "main", "feat/x", "wip/feat/x", false, runner)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Equal(t, "feat/x", checkedOut)
	})

	t.Run("should rebase WIP branch and return wip status for dirty repo", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		result := repo.RestoreAfterSync("/repo", "my-repo", "main", "feat/x", "wip/feat/x", true, runner)

		// then
		assert.Contains(t, result.Status, "synced (wip: wip/feat/x)")
	})
}

func TestRunSync(t *testing.T) {
	t.Parallel()

	t.Run("should report no repos when directory is empty", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunSync(root, runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no git repositories found")
	})

	t.Run("should sync repos and report summary", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		createGitRepo(t, root+"/repo-b")
		runner := doubles.NewGitRunnerStub().
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/origin/HEAD"},
				"refs/remotes/origin/main",
			).
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "")
		var buf bytes.Buffer

		// when
		err := repo.RunSync(root, runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "summary:")
		assert.Contains(t, buf.String(), "synced")
	})

	t.Run("should report failures in summary", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		runner := doubles.NewGitRunnerStub().
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/origin/HEAD"},
				"refs/remotes/origin/main",
			).
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "").
			WithRunError([]string{"fetch", "--all", "--prune", "-q"}, errors.New("network error"))
		var buf bytes.Buffer

		// when
		err := repo.RunSync(root, runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "FAIL")
		assert.Contains(t, buf.String(), "failed")
	})
}

func TestSyncSingleRepo(t *testing.T) {
	t.Parallel()

	t.Run("should sync clean repo successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/origin/HEAD"},
				"refs/remotes/origin/main",
			).
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "")

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Equal(t, "my-repo", result.Name)
		assert.Equal(t, "synced", result.Status)
	})

	t.Run("should preserve WIP state for dirty repo", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/origin/HEAD"},
				"refs/remotes/origin/main",
			).
			WithOutput([]string{"branch", "--show-current"}, "feat/x").
			WithOutput([]string{"status", "--porcelain"}, " M file.go")

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Equal(t, "my-repo", result.Name)
		assert.Contains(t, result.Status, "synced (wip:")
	})
}
