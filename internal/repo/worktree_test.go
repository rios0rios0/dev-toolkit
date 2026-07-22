package repo_test

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

const goneRefFormat = "--format=%(refname:short) %(upstream:track)"

// worktreeEntry renders one `git worktree list --porcelain` record.
func worktreeEntry(path, branch string, extra ...string) string {
	lines := []string{"worktree " + path, "HEAD abc123"}
	if branch == "" {
		lines = append(lines, "detached")
	} else {
		lines = append(lines, "branch refs/heads/"+branch)
	}
	lines = append(lines, extra...)
	return strings.Join(lines, "\n") + "\n"
}

// worktreeRunner builds a stub answering the commands ScanWorktrees issues.
func worktreeRunner(porcelain string) *doubles.GitRunnerStub {
	return doubles.NewGitRunnerStub().
		WithOutput([]string{"worktree", "list", "--porcelain"}, porcelain).
		WithOutput([]string{"symbolic-ref", "refs/remotes/origin/HEAD"}, "refs/remotes/origin/main").
		WithOutput([]string{"branch", "--merged", "main"}, "* main")
}

func TestParseWorktreeList(t *testing.T) {
	t.Parallel()

	t.Run("should mark the first entry as the main worktree", func(t *testing.T) {
		t.Parallel()
		// given
		output := worktreeEntry("/root/repo", "main") + "\n" + worktreeEntry("/root/repo-wt", "feat/x")

		// when
		worktrees := repo.ParseWorktreeList(output)

		// then
		require.Len(t, worktrees, 2)
		assert.True(t, worktrees[0].IsMain)
		assert.False(t, worktrees[1].IsMain)
	})

	t.Run("should strip the refs/heads prefix from branch names", func(t *testing.T) {
		t.Parallel()
		// given
		output := worktreeEntry("/root/repo", "feat/audit-log")

		// when
		worktrees := repo.ParseWorktreeList(output)

		// then
		require.Len(t, worktrees, 1)
		assert.Equal(t, "feat/audit-log", worktrees[0].Branch)
		assert.Equal(t, "abc123", worktrees[0].Head)
	})

	t.Run("should flag detached, locked, prunable and bare entries", func(t *testing.T) {
		t.Parallel()
		// given
		output := worktreeEntry("/root/repo", "main", "bare") + "\n" +
			worktreeEntry("/root/wt-detached", "") + "\n" +
			worktreeEntry("/root/wt-locked", "feat/x", "locked because reasons") + "\n" +
			worktreeEntry("/root/wt-gone", "feat/y", "prunable gitdir file points to non-existent location")

		// when
		worktrees := repo.ParseWorktreeList(output)

		// then
		require.Len(t, worktrees, 4)
		assert.True(t, worktrees[0].Bare)
		assert.True(t, worktrees[1].Detached)
		assert.True(t, worktrees[2].Locked)
		assert.True(t, worktrees[3].Prunable)
	})

	t.Run("should return nothing when output is empty", func(t *testing.T) {
		t.Parallel()
		// given
		output := ""

		// when
		worktrees := repo.ParseWorktreeList(output)

		// then
		assert.Empty(t, worktrees)
	})

	t.Run("should tolerate a missing blank line between records", func(t *testing.T) {
		t.Parallel()
		// given
		output := "worktree /root/repo\nHEAD abc\nworktree /root/repo-wt\nHEAD def\n"

		// when
		worktrees := repo.ParseWorktreeList(output)

		// then
		require.Len(t, worktrees, 2)
		assert.Equal(t, "/root/repo", worktrees[0].Path)
		assert.Equal(t, "/root/repo-wt", worktrees[1].Path)
	})
}

func TestScanWorktrees(t *testing.T) {
	t.Parallel()

	t.Run("should return nothing when the repo has only its main worktree", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(worktreeEntry("/root/repo", "main"))

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		assert.Empty(t, candidates)
	})

	t.Run("should always keep the main worktree", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main") + "\n" + worktreeEntry("/root/repo-wt", "feat/x"),
		)

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.True(t, candidates[0].Keep)
		assert.Equal(t, repo.WorktreeKeepMain, candidates[0].Reason)
		assert.Equal(t, "repo", candidates[0].Repo)
	})

	t.Run("should mark a prunable registration as stale", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main") + "\n" +
				worktreeEntry("/root/repo-wt", "feat/x", "prunable gitdir file points to non-existent location"),
		)

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.False(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeReasonStale, candidates[1].Reason)
	})

	t.Run("should mark a worktree living outside the root as orphaned", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main") + "\n" + worktreeEntry("/tmp/scratch/wt-stray", "feat/x"),
		)

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.False(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeReasonOrphaned, candidates[1].Reason)
	})

	t.Run("should mark a worktree whose branch is merged", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main")+"\n"+worktreeEntry("/root/repo-wt", "feat/done"),
		).WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done")

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.False(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeReasonMerged, candidates[1].Reason)
	})

	t.Run("should mark a worktree whose upstream was deleted", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main")+"\n"+worktreeEntry("/root/repo-wt", "feat/shipped"),
		).WithOutput(
			[]string{"for-each-ref", goneRefFormat, "refs/heads"},
			"main [behind 1]\nfeat/shipped [gone]",
		)

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.False(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeReasonGone, candidates[1].Reason)
	})

	t.Run("should not treat a worktree as orphaned when the root is relative", func(t *testing.T) {
		t.Parallel()
		// given git always reports absolute worktree paths, while the root may be relative
		cwd, err := os.Getwd()
		require.NoError(t, err)
		runner := worktreeRunner(
			worktreeEntry(cwd+"/repo", "main") + "\n" + worktreeEntry(cwd+"/repo-wt", "feat/wip"),
		)

		// when
		candidates := repo.ScanWorktrees("repo", ".", runner)

		// then
		require.Len(t, candidates, 2)
		assert.True(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeKeepActive, candidates[1].Reason)
	})

	t.Run("should still flag a worktree outside a relative root as orphaned", func(t *testing.T) {
		t.Parallel()
		// given
		cwd, err := os.Getwd()
		require.NoError(t, err)
		runner := worktreeRunner(
			worktreeEntry(cwd+"/repo", "main") + "\n" + worktreeEntry("/tmp/scratch/wt-stray", "feat/wip"),
		)

		// when
		candidates := repo.ScanWorktrees("repo", ".", runner)

		// then
		require.Len(t, candidates, 2)
		assert.False(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeReasonOrphaned, candidates[1].Reason)
	})

	t.Run("should keep an active worktree that matches no removal rule", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main") + "\n" + worktreeEntry("/root/repo-wt", "feat/wip"),
		)

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.True(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeKeepActive, candidates[1].Reason)
	})

	t.Run("should keep a locked worktree even when its branch is merged", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main")+"\n"+
				worktreeEntry("/root/repo-wt", "feat/done", "locked in use"),
		).WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done")

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.True(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeKeepLocked, candidates[1].Reason)
	})

	t.Run("should keep a dirty worktree even when its branch is merged", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main")+"\n"+worktreeEntry("/root/repo-wt", "feat/done"),
		).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done").
			WithOutputForDir("/root/repo-wt", []string{"status", "--porcelain"}, " M internal/repo/clone.go")

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.True(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeKeepDirty, candidates[1].Reason)
	})

	t.Run("should keep a worktree holding unpushed commits", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main")+"\n"+worktreeEntry("/root/repo-wt", "feat/done"),
		).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done").
			WithOutputForDir("/root/repo-wt", []string{"rev-list", "--count", "@{upstream}..HEAD"}, "3")

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.True(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeKeepUnpushed, candidates[1].Reason)
	})

	t.Run("should not treat an in-sync branch as holding unpushed commits", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main")+"\n"+worktreeEntry("/root/repo-wt", "feat/done"),
		).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done").
			WithOutputForDir("/root/repo-wt", []string{"rev-list", "--count", "@{upstream}..HEAD"}, "0")

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.False(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeReasonMerged, candidates[1].Reason)
	})

	t.Run("should keep a detached worktree", func(t *testing.T) {
		t.Parallel()
		// given
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main") + "\n" + worktreeEntry("/root/repo-wt", ""),
		)

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.True(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeKeepDetached, candidates[1].Reason)
	})

	t.Run("should not treat a gone branch as holding unpushed commits", func(t *testing.T) {
		t.Parallel()
		// given a gone upstream makes `rev-list @{upstream}..HEAD` meaningless
		runner := worktreeRunner(
			worktreeEntry("/root/repo", "main")+"\n"+worktreeEntry("/root/repo-wt", "feat/shipped"),
		).
			WithOutput([]string{"for-each-ref", goneRefFormat, "refs/heads"}, "feat/shipped [gone]").
			WithOutputForDir("/root/repo-wt", []string{"rev-list", "--count", "@{upstream}..HEAD"}, "7")

		// when
		candidates := repo.ScanWorktrees("/root/repo", "/root", runner)

		// then
		require.Len(t, candidates, 2)
		assert.False(t, candidates[1].Keep)
		assert.Equal(t, repo.WorktreeReasonGone, candidates[1].Reason)
	})
}

func TestListGoneBranches(t *testing.T) {
	t.Parallel()

	t.Run("should return only branches whose upstream is gone", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub().WithOutput(
			[]string{"for-each-ref", goneRefFormat, "refs/heads"},
			"main \nfeat/a [gone]\nfeat/b [ahead 2]\nfeat/c [gone]",
		)

		// when
		gone := repo.ListGoneBranches("/root/repo", runner)

		// then
		assert.Len(t, gone, 2)
		assert.Contains(t, gone, "feat/a")
		assert.Contains(t, gone, "feat/c")
	})

	t.Run("should return an empty set when there is no output", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewGitRunnerStub()

		// when
		gone := repo.ListGoneBranches("/root/repo", runner)

		// then
		assert.Empty(t, gone)
	})
}

func TestFilterRemovableWorktrees(t *testing.T) {
	t.Parallel()

	t.Run("should return only candidates not marked to keep", func(t *testing.T) {
		t.Parallel()
		// given
		candidates := []repo.WorktreeCandidate{
			{Repo: "a", Keep: true, Reason: repo.WorktreeKeepMain},
			{Repo: "b", Reason: repo.WorktreeReasonMerged},
			{Repo: "c", Keep: true, Reason: repo.WorktreeKeepDirty},
			{Repo: "d", Reason: repo.WorktreeReasonStale},
		}

		// when
		removable := repo.FilterRemovableWorktrees(candidates)

		// then
		require.Len(t, removable, 2)
		assert.Equal(t, "b", removable[0].Repo)
		assert.Equal(t, "d", removable[1].Repo)
	})

	t.Run("should return nothing when every candidate is kept", func(t *testing.T) {
		t.Parallel()
		// given
		candidates := []repo.WorktreeCandidate{{Repo: "a", Keep: true}}

		// when
		removable := repo.FilterRemovableWorktrees(candidates)

		// then
		assert.Empty(t, removable)
	})
}

func TestRemoveWorktree(t *testing.T) {
	t.Parallel()

	t.Run("should drop a stale registration with worktree prune", func(t *testing.T) {
		t.Parallel()
		// given a runner that fails only on `worktree prune`
		candidate := repo.WorktreeCandidate{
			RepoPath: "/root/repo",
			Worktree: repo.Worktree{Path: "/root/repo-wt"},
			Reason:   repo.WorktreeReasonStale,
		}
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"worktree", "prune"}, errors.New("boom"))

		// when
		err := repo.RemoveWorktree(candidate, runner)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "worktree prune")
	})

	t.Run("should not call worktree remove for a stale registration", func(t *testing.T) {
		t.Parallel()
		// given a runner that fails only on `worktree remove`
		candidate := repo.WorktreeCandidate{
			RepoPath: "/root/repo",
			Worktree: repo.Worktree{Path: "/root/repo-wt"},
			Reason:   repo.WorktreeReasonStale,
		}
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"worktree", "remove", "/root/repo-wt"}, errors.New("boom"))

		// when
		err := repo.RemoveWorktree(candidate, runner)

		// then
		require.NoError(t, err)
	})

	t.Run("should remove a live worktree through git", func(t *testing.T) {
		t.Parallel()
		// given a runner that fails only on the exact `worktree remove <path>` call
		candidate := repo.WorktreeCandidate{
			RepoPath: "/root/repo",
			Worktree: repo.Worktree{Path: "/root/repo-wt"},
			Reason:   repo.WorktreeReasonMerged,
		}
		runner := doubles.NewGitRunnerStub().
			WithRunError([]string{"worktree", "remove", "/root/repo-wt"}, errors.New("contains modified files"))

		// when
		err := repo.RemoveWorktree(candidate, runner)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "worktree remove")
		assert.Contains(t, err.Error(), "contains modified files")
	})

	t.Run("should succeed when git accepts the removal", func(t *testing.T) {
		t.Parallel()
		// given
		candidate := repo.WorktreeCandidate{
			RepoPath: "/root/repo",
			Worktree: repo.Worktree{Path: "/root/repo-wt"},
			Reason:   repo.WorktreeReasonMerged,
		}
		runner := doubles.NewGitRunnerStub()

		// when
		err := repo.RemoveWorktree(candidate, runner)

		// then
		require.NoError(t, err)
	})
}

func TestPromptRemoveWorktree(t *testing.T) {
	t.Parallel()

	t.Run("should confirm when the user answers y", func(t *testing.T) {
		t.Parallel()
		// given
		candidate := repo.WorktreeCandidate{
			Repo:     "repo",
			Worktree: repo.Worktree{Path: "/root/repo-wt", Branch: "feat/x"},
			Reason:   repo.WorktreeReasonMerged,
		}
		scanner := bufio.NewScanner(strings.NewReader("y\n"))
		var buf bytes.Buffer

		// when
		confirmed := repo.PromptRemoveWorktree(candidate, scanner, &buf)

		// then
		assert.True(t, confirmed)
		assert.Contains(t, buf.String(), "/root/repo-wt")
		assert.Contains(t, buf.String(), repo.WorktreeReasonMerged)
	})

	t.Run("should decline when the user answers n", func(t *testing.T) {
		t.Parallel()
		// given
		candidate := repo.WorktreeCandidate{Worktree: repo.Worktree{Path: "/root/repo-wt", Branch: "feat/x"}}
		scanner := bufio.NewScanner(strings.NewReader("n\n"))
		var buf bytes.Buffer

		// when
		confirmed := repo.PromptRemoveWorktree(candidate, scanner, &buf)

		// then
		assert.False(t, confirmed)
	})

	t.Run("should read one answer per prompt from a shared scanner", func(t *testing.T) {
		t.Parallel()
		// given
		first := repo.WorktreeCandidate{Worktree: repo.Worktree{Path: "/root/wt-a", Branch: "feat/a"}}
		second := repo.WorktreeCandidate{Worktree: repo.Worktree{Path: "/root/wt-b", Branch: "feat/b"}}
		scanner := bufio.NewScanner(strings.NewReader("n\ny\n"))
		var buf bytes.Buffer

		// when
		firstAnswer := repo.PromptRemoveWorktree(first, scanner, &buf)
		secondAnswer := repo.PromptRemoveWorktree(second, scanner, &buf)

		// then
		assert.False(t, firstAnswer)
		assert.True(t, secondAnswer)
	})

	t.Run("should fall back to the head when the worktree is detached", func(t *testing.T) {
		t.Parallel()
		// given
		candidate := repo.WorktreeCandidate{Worktree: repo.Worktree{Path: "/root/wt", Head: "abc123"}}
		scanner := bufio.NewScanner(strings.NewReader("\n"))
		var buf bytes.Buffer

		// when
		repo.PromptRemoveWorktree(candidate, scanner, &buf)

		// then
		assert.Contains(t, buf.String(), "abc123")
	})
}

func TestRunWorktreePrune(t *testing.T) {
	t.Parallel()

	t.Run("should warn when no repositories exist", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		cfg := repo.WorktreePruneConfig{
			RootDir: t.TempDir(),
			Runner:  doubles.NewGitRunnerStub(),
			Output:  &buf,
		}

		// when
		err := repo.RunWorktreePrune(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no git repositories found")
	})

	t.Run("should report when nothing is disposable", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		var buf bytes.Buffer
		cfg := repo.WorktreePruneConfig{
			RootDir: root,
			Runner:  worktreeRunner(worktreeEntry(root+"/repo-a", "main")),
			Output:  &buf,
		}

		// when
		err := repo.RunWorktreePrune(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no disposable worktrees found")
	})

	t.Run("should report would-remove in dry-run mode without removing", func(t *testing.T) {
		t.Parallel()
		// given a runner that errors if any removal is attempted
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		porcelain := worktreeEntry(root+"/repo-a", "main") + "\n" +
			worktreeEntry(root+"/repo-a-wt", "feat/done")
		runner := worktreeRunner(porcelain).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done").
			WithRunError([]string{"worktree", "remove", root + "/repo-a-wt"}, errors.New("must not run"))
		var buf bytes.Buffer
		cfg := repo.WorktreePruneConfig{RootDir: root, DryRun: true, Runner: runner, Output: &buf}

		// when
		err := repo.RunWorktreePrune(cfg)

		// then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "dry-run mode enabled")
		assert.Contains(t, output, "would remove worktree")
		assert.Contains(t, output, "removed=0")
	})

	t.Run("should remove disposable worktrees when confirmation is assumed", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		porcelain := worktreeEntry(root+"/repo-a", "main") + "\n" +
			worktreeEntry(root+"/repo-a-wt", "feat/done")
		runner := worktreeRunner(porcelain).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done")
		var buf bytes.Buffer
		cfg := repo.WorktreePruneConfig{RootDir: root, AssumeYes: true, Runner: runner, Output: &buf}

		// when
		err := repo.RunWorktreePrune(cfg)

		// then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "removed worktree")
		assert.Contains(t, output, "removed=1")
		assert.Contains(t, output, "failed=0")
	})

	t.Run("should count a failed removal instead of aborting", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		porcelain := worktreeEntry(root+"/repo-a", "main") + "\n" +
			worktreeEntry(root+"/repo-a-wt", "feat/done")
		runner := worktreeRunner(porcelain).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done").
			WithRunError(
				[]string{"worktree", "remove", root + "/repo-a-wt"},
				errors.New("contains modified files"),
			)
		var buf bytes.Buffer
		cfg := repo.WorktreePruneConfig{RootDir: root, AssumeYes: true, Runner: runner, Output: &buf}

		// when
		err := repo.RunWorktreePrune(cfg)

		// then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "could not remove worktree")
		assert.Contains(t, output, "failed=1")
		assert.Contains(t, output, "removed=0")
	})

	t.Run("should keep disposable worktrees when input is not a terminal", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		porcelain := worktreeEntry(root+"/repo-a", "main") + "\n" +
			worktreeEntry(root+"/repo-a-wt", "feat/done")
		runner := worktreeRunner(porcelain).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done").
			WithRunError([]string{"worktree", "remove", root + "/repo-a-wt"}, errors.New("must not run"))
		var buf bytes.Buffer
		cfg := repo.WorktreePruneConfig{
			RootDir: root,
			Runner:  runner,
			Output:  &buf,
			Input:   strings.NewReader("y\n"),
		}

		// when
		err := repo.RunWorktreePrune(cfg)

		// then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "non-interactive")
		assert.Contains(t, output, "removed=0")
		assert.Contains(t, output, "kept=1")
	})

	t.Run("should report a protected worktree so it does not disappear silently", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		porcelain := worktreeEntry(root+"/repo-a", "main") + "\n" +
			worktreeEntry(root+"/repo-a-wt", "feat/done")
		runner := worktreeRunner(porcelain).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done").
			WithOutputForDir(root+"/repo-a-wt", []string{"status", "--porcelain"}, " M main.go")
		var buf bytes.Buffer
		cfg := repo.WorktreePruneConfig{RootDir: root, AssumeYes: true, Runner: runner, Output: &buf}

		// when
		err := repo.RunWorktreePrune(cfg)

		// then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "kept worktree")
		assert.Contains(t, output, repo.WorktreeKeepDirty)
		assert.Contains(t, output, "no disposable worktrees found")
	})
}

func TestRunWorktreeList(t *testing.T) {
	t.Parallel()

	t.Run("should warn when no repositories exist", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer

		// when
		err := repo.RunWorktreeList(t.TempDir(), doubles.NewGitRunnerStub(), &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no git repositories found")
	})

	t.Run("should list linked worktrees with their classification", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		porcelain := worktreeEntry(root+"/repo-a", "main") + "\n" +
			worktreeEntry(root+"/repo-a-wt", "feat/done")
		runner := worktreeRunner(porcelain).
			WithOutput([]string{"branch", "--merged", "main"}, "* main\n+ feat/done")
		var buf bytes.Buffer

		// when
		err := repo.RunWorktreeList(root, runner, &buf)

		// then
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "repo-a-wt")
		assert.Contains(t, output, repo.WorktreeReasonMerged)
		assert.Contains(t, output, "worktrees=1")
		assert.Contains(t, output, "removable=1")
	})

	t.Run("should not list the main worktree as an entry", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		runner := worktreeRunner(
			worktreeEntry(root+"/repo-a", "main") + "\n" + worktreeEntry(root+"/repo-a-wt", "feat/x"),
		)
		var buf bytes.Buffer

		// when
		err := repo.RunWorktreeList(root, runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "worktrees=1")
	})
}
