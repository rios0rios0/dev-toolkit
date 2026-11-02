package repo_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

// recordRunCalls makes runner remember every command it runs, joined with spaces, while
// still answering them through the behavior already configured.
func recordRunCalls(runner *doubles.GitRunnerStub) *[]string {
	var calls []string
	answer := runner.RunFunc
	runner.RunFunc = func(dir string, args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		return answer(dir, args...)
	}
	return &calls
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
			WithRef("refs/heads/main").
			WithRunError([]string{"checkout", "main"}, errors.New("checkout failed"))

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "main", "feat/x", "wip/feat/x", false, runner)

		// then
		assert.Contains(t, result.Status, "FAIL (checkout main")
	})

	t.Run("should create the default branch from origin when it was never checked out locally", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().WithRef("refs/remotes/origin/develop")
		calls := recordRunCalls(runner)

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "develop", "master", "wip/master", false, runner)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Equal(t, []string{"checkout --track origin/develop", "pull --rebase"}, *calls)
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

	t.Run("should stay on default branch when starting from non-default branch", func(t *testing.T) {
		t.Parallel()
		// given
		var checkouts []string
		runner := doubles.NewGitRunnerStub().WithRef("refs/heads/main")
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) > 0 && args[0] == "checkout" {
				checkouts = append(checkouts, args[1])
			}
			return nil
		}

		// when
		result := repo.SyncAndRestore("/repo", "my-repo", "main", "feat/x", "wip/feat/x", false, runner)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Equal(t, []string{"main"}, checkouts)
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

func TestFinalizeSync(t *testing.T) {
	t.Parallel()

	t.Run("should return synced for clean repo on default branch", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		result := repo.FinalizeSync("/repo", "my-repo", "main", "wip/main", false, runner)

		// then
		assert.Equal(t, "synced", result.Status)
	})

	t.Run("should stay on default branch when clean repo was on non-default branch", func(t *testing.T) {
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
		result := repo.FinalizeSync("/repo", "my-repo", "main", "wip/feat/x", false, runner)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Empty(t, checkedOut)
	})

	t.Run("should rebase WIP branch and checkout default branch for dirty repo", func(t *testing.T) {
		t.Parallel()
		// given
		var lastCheckedOut string
		runner := doubles.NewGitRunnerStub()
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) > 0 && args[0] == "checkout" {
				lastCheckedOut = args[1]
			}
			return nil
		}

		// when
		result := repo.FinalizeSync("/repo", "my-repo", "main", "wip/feat/x", true, runner)

		// then
		assert.Contains(t, result.Status, "synced (wip: wip/feat/x)")
		assert.Equal(t, "main", lastCheckedOut)
	})

	t.Run("should return FAIL when checkout to default branch fails for dirty repo", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) >= 2 && args[0] == "checkout" && args[1] == "main" {
				return errors.New("checkout failed")
			}
			return nil
		}

		// when
		result := repo.FinalizeSync("/repo", "my-repo", "main", "wip/feat/x", true, runner)

		// then
		assert.Contains(t, result.Status, "FAIL")
		assert.Contains(t, result.Status, "checkout main")
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
			WithRemoteHEAD("origin", "main").
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "")
		var buf bytes.Buffer

		// when
		err := repo.RunSync(root, runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "summary")
		assert.Contains(t, buf.String(), "synced")
	})

	t.Run("should report failures in summary", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
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
			WithRemoteHEAD("origin", "main").
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
			WithRemoteHEAD("origin", "main").
			WithOutput([]string{"branch", "--show-current"}, "feat/x").
			WithOutput([]string{"status", "--porcelain"}, " M file.go")

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Equal(t, "my-repo", result.Name)
		assert.Contains(t, result.Status, "synced (wip:")
	})

	t.Run("should fetch and refresh origin/HEAD before switching to the default branch", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithRef("refs/heads/main").
			WithOutput([]string{"branch", "--show-current"}, "main")
		calls := recordRunCalls(runner)

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Equal(t, []string{
			"fetch --all --prune -q",
			"remote set-head origin --auto",
			"checkout main",
			"pull --rebase",
		}, *calls)
	})

	t.Run("should sync the branch the remote reports when origin/HEAD was never recorded", func(t *testing.T) {
		t.Parallel()
		// given: a leftover local main must not be mistaken for the default branch
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"ls-remote", "--symref", "origin", "HEAD"}, advertisedHEAD("develop")).
			WithRef("refs/heads/develop").
			WithRef("refs/heads/main").
			WithOutput([]string{"branch", "--show-current"}, "feat/x")
		calls := recordRunCalls(runner)

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Contains(t, *calls, "checkout develop")
		assert.NotContains(t, *calls, "checkout main")
	})

	t.Run("should follow a renamed default branch instead of the deleted one origin/HEAD names", func(t *testing.T) {
		t.Parallel()
		// given: the remote renamed master to main, the pruning fetch removed origin/master
		// while origin/HEAD still names it, and main was never checked out here
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/master").
			WithOutput([]string{"ls-remote", "--symref", "origin", "HEAD"}, advertisedHEAD("main")).
			WithRef("refs/heads/master").
			WithRef("refs/remotes/origin/main").
			WithOutput([]string{"branch", "--show-current"}, "master")
		calls := recordRunCalls(runner)

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Equal(t, "synced", result.Status)
		assert.Contains(t, *calls, "checkout --track origin/main")
		assert.NotContains(t, *calls, "checkout master")
	})

	t.Run("should fail without touching the repository when the default branch is unknown", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"branch", "--show-current"}, "feat/x").
			WithOutput([]string{"status", "--porcelain"}, " M file.go")
		calls := recordRunCalls(runner)

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Equal(t, "FAIL (default branch not found for remote origin)", result.Status)
		assert.Equal(t, []string{"fetch --all --prune -q", "remote set-head origin --auto"}, *calls)
	})

	t.Run("should leave a dirty repository untouched when fetch fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().
			WithRemoteHEAD("origin", "main").
			WithOutput([]string{"branch", "--show-current"}, "feat/x").
			WithOutput([]string{"status", "--porcelain"}, " M file.go").
			WithRunError([]string{"fetch", "--all", "--prune", "-q"}, errors.New("network error"))
		calls := recordRunCalls(runner)

		// when
		result := repo.SyncSingleRepo("/root/my-repo", "/root", runner)

		// then
		assert.Contains(t, result.Status, "FAIL (fetch")
		assert.Equal(t, []string{"fetch --all --prune -q"}, *calls)
	})
}
