package gist_test

import (
	"bytes"
	"errors"
	"testing"

	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/gist"
	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestComputeDiff(t *testing.T) {
	t.Parallel()

	t.Run("should return missing entries for remote gists not present locally", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []gist.Gist{
			{ID: "1", Owner: "alice", Description: "First"},
			{ID: "2", Owner: "alice", Description: "Second"},
		}
		local := []string{"first"}

		// when
		missing, extra := gist.ComputeDiff(remote, local)

		// then
		require.Len(t, missing, 1)
		assert.Equal(t, "2", missing[0].ID)
		assert.Empty(t, extra)
	})

	t.Run("should return extras for local gists not present on remote", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []gist.Gist{{ID: "1", Owner: "alice", Description: "First"}}
		local := []string{"first", "orphan"}

		// when
		missing, extra := gist.ComputeDiff(remote, local)

		// then
		assert.Empty(t, missing)
		assert.Equal(t, []string{"orphan"}, extra)
	})

	t.Run("should report nothing when remote and local match", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []gist.Gist{{ID: "1", Owner: "alice", Description: "Same"}}
		local := []string{"same"}

		// when
		missing, extra := gist.ComputeDiff(remote, local)

		// then
		assert.Empty(t, missing)
		assert.Empty(t, extra)
	})

	t.Run("should disambiguate colliding slugs and not drop entries", func(t *testing.T) {
		t.Parallel()
		// given two remote gists with the same description (and so the same
		// natural slug) — both must appear as missing instead of one
		// overwriting the other in the remote map.
		remote := []gist.Gist{
			{ID: "aaaaaaa1", Owner: "alice", Description: "Duplicate"},
			{ID: "bbbbbbb2", Owner: "alice", Description: "Duplicate"},
		}

		// when
		missing, extra := gist.ComputeDiff(remote, nil)

		// then
		assert.Len(t, missing, 2)
		assert.Empty(t, extra)
	})
}

func TestCloneMissing(t *testing.T) {
	t.Parallel()

	noopPreflight := func(_ string, _ logger.FieldLogger) error { return nil }

	t.Run("should return zero counts when nothing is missing", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		cfg := gist.CloneConfig{
			RootDir: "/tmp/gists",
			Output:  &buf,
			Logger:  repo.NewLogger(&buf),
		}

		// when
		cloned, failed := gist.CloneMissing(nil, cfg)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 0, failed)
	})

	t.Run("should clone every gist when preflight succeeds", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		cfg := gist.CloneConfig{
			RootDir:   "/tmp/gists",
			SSHAlias:  "mine",
			Runner:    doubles.NewGitRunnerStub(),
			Output:    &buf,
			Logger:    repo.NewLogger(&buf),
			Preflight: noopPreflight,
		}
		missing := []gist.Gist{
			{ID: "1", Owner: "alice", Description: "First"},
			{ID: "2", Owner: "alice", Description: "Second"},
		}

		// when
		cloned, failed := gist.CloneMissing(missing, cfg)

		// then
		assert.Equal(t, 2, cloned)
		assert.Equal(t, 0, failed)
		assert.Contains(t, buf.String(), "gist cloned")
	})

	t.Run("should mark every gist as failed when preflight fails", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		failPreflight := func(_ string, _ logger.FieldLogger) error { return errors.New("ssh failed") }
		cfg := gist.CloneConfig{
			RootDir:   "/tmp/gists",
			SSHAlias:  "mine",
			Output:    &buf,
			Logger:    repo.NewLogger(&buf),
			Preflight: failPreflight,
		}
		missing := []gist.Gist{
			{ID: "1", Owner: "alice"},
			{ID: "2", Owner: "alice"},
		}

		// when
		cloned, failed := gist.CloneMissing(missing, cfg)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 2, failed)
		assert.Contains(t, buf.String(), "SSH preflight failed")
	})

	t.Run("should log preview output without cloning in dry-run mode", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		cfg := gist.CloneConfig{
			RootDir:  "/tmp/gists",
			SSHAlias: "mine",
			DryRun:   true,
			Output:   &buf,
			Logger:   repo.NewLogger(&buf),
		}
		missing := []gist.Gist{{ID: "1", Owner: "alice", Description: "Snippet"}}

		// when
		cloned, failed := gist.CloneMissing(missing, cfg)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 0, failed)
		assert.Contains(t, buf.String(), "would clone gist")
	})
}

func TestRunClone(t *testing.T) {
	t.Parallel()

	t.Run("should report sync when remote and local are aligned", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		root := t.TempDir()
		createGistRepo(t, root+"/snippet")
		provider := doubles.NewGistProviderStub().WithGists([]gist.Gist{
			{ID: "abc", Owner: "alice", Description: "Snippet"},
		})
		cfg := gist.CloneConfig{
			RootDir:  root,
			Owner:    "alice",
			SSHAlias: "mine",
			Provider: provider,
			Runner:   doubles.NewGitRunnerStub(),
			Output:   &buf,
			Logger:   repo.NewLogger(&buf),
		}

		// when
		err := gist.RunClone(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "everything is in sync")
	})

	t.Run("should return an error when the provider call fails", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		provider := doubles.NewGistProviderStub().WithListError(errors.New("API error"))
		cfg := gist.CloneConfig{
			RootDir:  "/tmp/gists",
			Owner:    "alice",
			SSHAlias: "mine",
			Provider: provider,
			Output:   &buf,
			Logger:   repo.NewLogger(&buf),
		}

		// when
		err := gist.RunClone(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to discover gists")
	})

	t.Run("should return an error when the owner is missing", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		cfg := gist.CloneConfig{
			RootDir: "/tmp/gists",
			Output:  &buf,
			Logger:  repo.NewLogger(&buf),
		}

		// when
		err := gist.RunClone(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner")
	})

	t.Run("should preview missing gists in dry-run mode", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		root := t.TempDir()
		provider := doubles.NewGistProviderStub().WithGists([]gist.Gist{
			{ID: "abc", Owner: "alice", Description: "Brand New"},
		})
		cfg := gist.CloneConfig{
			RootDir:  root,
			Owner:    "alice",
			SSHAlias: "mine",
			DryRun:   true,
			Provider: provider,
			Output:   &buf,
			Logger:   repo.NewLogger(&buf),
		}

		// when
		err := gist.RunClone(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "would clone gist")
	})
}

func TestParallelClone(t *testing.T) {
	t.Parallel()

	t.Run("should clone every gist successfully", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		runner := doubles.NewGitRunnerStub()
		gists := []gist.Gist{
			{ID: "1", Owner: "alice", Description: "First"},
			{ID: "2", Owner: "alice", Description: "Second"},
		}

		// when
		cloned, failed := gist.ParallelClone(gists, "mine", "/tmp/gists", runner, repo.NewLogger(&buf))

		// then
		assert.Equal(t, 2, cloned)
		assert.Equal(t, 0, failed)
	})

	t.Run("should count failures when the runner returns an error", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		runner := doubles.NewGitRunnerStub().WithCloneError(errors.New("permission denied"))
		gists := []gist.Gist{
			{ID: "1", Owner: "alice", Description: "First"},
			{ID: "2", Owner: "alice", Description: "Second"},
		}

		// when
		cloned, failed := gist.ParallelClone(gists, "mine", "/tmp/gists", runner, repo.NewLogger(&buf))

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 2, failed)
		assert.Contains(t, buf.String(), "clone failed")
	})

	t.Run("should be safe with an empty input", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		runner := doubles.NewGitRunnerStub()

		// when
		cloned, failed := gist.ParallelClone(nil, "mine", "/tmp/gists", runner, repo.NewLogger(&buf))

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 0, failed)
	})
}

func TestMaxCloneArgs(t *testing.T) {
	t.Parallel()

	t.Run("should return 2", func(t *testing.T) {
		t.Parallel()
		// given / when
		result := gist.MaxCloneArgs()

		// then
		assert.Equal(t, 2, result)
	})
}
