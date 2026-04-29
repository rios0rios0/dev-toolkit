package gist_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/gist"
)

func createGistRepo(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(filepath.Join(path, ".git"), 0o750)
	assert.NoError(t, err)
}

func TestScanLocalGists(t *testing.T) {
	t.Parallel()

	t.Run("should return owner/slug entries for nested gists", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGistRepo(t, filepath.Join(root, "alice", "snippet-one"))
		createGistRepo(t, filepath.Join(root, "alice", "snippet-two"))
		createGistRepo(t, filepath.Join(root, "bob", "config"))

		// when
		gists := gist.ScanLocalGists(root)

		// then
		assert.ElementsMatch(t, []string{
			"alice/snippet-one",
			"alice/snippet-two",
			"bob/config",
		}, gists)
	})

	t.Run("should ignore directories without a .git child", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGistRepo(t, filepath.Join(root, "alice", "real-gist"))
		err := os.MkdirAll(filepath.Join(root, "alice", "not-a-gist"), 0o750)
		require.NoError(t, err)

		// when
		gists := gist.ScanLocalGists(root)

		// then
		assert.Equal(t, []string{"alice/real-gist"}, gists)
	})

	t.Run("should return an empty slice when the root does not exist", func(t *testing.T) {
		t.Parallel()
		// given
		nonExistent := filepath.Join(t.TempDir(), "missing")

		// when
		gists := gist.ScanLocalGists(nonExistent)

		// then
		assert.Empty(t, gists)
	})
}
