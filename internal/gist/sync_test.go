package gist_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/gist"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunSync(t *testing.T) {
	t.Parallel()

	t.Run("should warn when no gists are found under the root", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := gist.RunSync(root, runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no gist repositories found")
	})

	t.Run("should sync every nested gist and report a summary", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGistRepo(t, root+"/snippet-one")
		createGistRepo(t, root+"/snippet-two")
		runner := doubles.NewGitRunnerStub().
			WithOutput(
				[]string{"symbolic-ref", "refs/remotes/origin/HEAD"},
				"refs/remotes/origin/main",
			).
			WithOutput([]string{"branch", "--show-current"}, "main").
			WithOutput([]string{"status", "--porcelain"}, "")
		var buf bytes.Buffer

		// when
		err := gist.RunSync(root, runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "summary")
		assert.Contains(t, buf.String(), "synced")
	})
}
