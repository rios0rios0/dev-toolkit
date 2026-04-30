package gist_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/gist"
)

func createGistRepo(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(filepath.Join(path, ".git"), 0o750)
	assert.NoError(t, err)
}

func TestScanLocalGists(t *testing.T) {
	t.Parallel()

	t.Run("should return slug entries one level under the root", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGistRepo(t, filepath.Join(root, "snippet-one"))
		createGistRepo(t, filepath.Join(root, "snippet-two"))
		createGistRepo(t, filepath.Join(root, "config"))

		// when
		gists := gist.ScanLocalGists(root)

		// then
		assert.ElementsMatch(t, []string{
			"snippet-one",
			"snippet-two",
			"config",
		}, gists)
	})

	t.Run("should ignore directories without a .git child", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGistRepo(t, filepath.Join(root, "real-gist"))
		err := os.MkdirAll(filepath.Join(root, "not-a-gist"), 0o750)
		require.NoError(t, err)

		// when
		gists := gist.ScanLocalGists(root)

		// then
		assert.Equal(t, []string{"real-gist"}, gists)
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
