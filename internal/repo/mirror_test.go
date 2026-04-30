package repo_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestRunMirror(t *testing.T) {
	t.Parallel()

	t.Run("should report no repos when directory is empty", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		var buf bytes.Buffer

		// when
		err := repo.RunMirror(repo.MirrorConfig{
			SourceDir:      root,
			TargetProvider: doubles.NewMirrorProviderStub(),
			Runner:         doubles.NewGitRunnerStub(),
			Output:         repo.NewLogger(&buf),
		})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not detect provider")
	})

	t.Run("should return error when source is not GitHub", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createDirStructure(t, root, "dev.azure.com/org")
		var buf bytes.Buffer

		// when
		err := repo.RunMirror(repo.MirrorConfig{
			SourceDir:      root + "/dev.azure.com/org",
			TargetProvider: doubles.NewMirrorProviderStub(),
			Runner:         doubles.NewGitRunnerStub(),
			Output:         repo.NewLogger(&buf),
		})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mirror only supports GitHub")
	})

	t.Run("should skip repos with existing codeberg remote", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createDirStructure(t, root, "github.com/owner")
		sourceDir := root + "/github.com/owner"
		createGitRepo(t, sourceDir+"/repo-a")
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"remote", "get-url", "codeberg"}, "git@codeberg.org:owner/repo-a.git")
		var buf bytes.Buffer

		// when
		err := repo.RunMirror(repo.MirrorConfig{
			SourceDir:      sourceDir,
			TargetProvider: doubles.NewMirrorProviderStub(),
			Runner:         runner,
			Output:         repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "skipped (remote exists)")
	})

	t.Run("should report failure when migration fails", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createDirStructure(t, root, "github.com/owner")
		sourceDir := root + "/github.com/owner"
		createGitRepo(t, sourceDir+"/repo-a")
		runner := doubles.NewGitRunnerStub()
		target := doubles.NewMirrorProviderStub().WithMigrateError(errors.New("API error"))
		var buf bytes.Buffer

		// when
		err := repo.RunMirror(repo.MirrorConfig{
			SourceDir:      sourceDir,
			TargetProvider: target,
			Runner:         runner,
			Output:         repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "FAIL (API error)")
	})

	t.Run("should log repos in dry-run mode without creating mirrors", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createDirStructure(t, root, "github.com/owner")
		sourceDir := root + "/github.com/owner"
		createGitRepo(t, sourceDir+"/repo-a")
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunMirror(repo.MirrorConfig{
			SourceDir:      sourceDir,
			DryRun:         true,
			TargetProvider: doubles.NewMirrorProviderStub(),
			Runner:         runner,
			Output:         repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "would create mirror")
	})

	t.Run("should mirror repos and add codeberg remote", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createDirStructure(t, root, "github.com/owner")
		sourceDir := root + "/github.com/owner"
		createGitRepo(t, sourceDir+"/repo-a")
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunMirror(repo.MirrorConfig{
			SourceDir:      sourceDir,
			SSHAlias:       "mine",
			TargetProvider: doubles.NewMirrorProviderStub(),
			Runner:         runner,
			Output:         repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "mirrored")
	})
}

func createDirStructure(t *testing.T, root, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, path), 0o750))
}
