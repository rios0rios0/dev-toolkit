package repo_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestRunRestore(t *testing.T) {
	t.Parallel()

	t.Run("should report no repos when directory is empty", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunRestore(repo.RestoreConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no git repositories found")
	})

	t.Run("should restore repos in failover state", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))

		var renamedPairs [][]string
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"remote", "get-url", "github"}, "git@github.com:owner/repo-a.git")
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) >= 4 && args[0] == "remote" && args[1] == "rename" {
				renamedPairs = append(renamedPairs, []string{args[2], args[3]})
			}
			return nil
		}
		var buf bytes.Buffer

		// when
		err := repo.RunRestore(repo.RestoreConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "restored")
		assert.Equal(t, [][]string{{"origin", "codeberg"}, {"github", "origin"}}, renamedPairs)
	})

	t.Run("should skip repos not in failover state", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunRestore(repo.RestoreConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "not in failover state")
	})

	t.Run("should continue restore even when push to github fails", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"remote", "get-url", "github"}, "git@github.com:owner/repo-a.git").
			WithRunError([]string{"push", "github", "--all", "--tags"}, errors.New("network error"))
		var buf bytes.Buffer

		// when
		err := repo.RunRestore(repo.RestoreConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "restored (push failed:")
	})

	t.Run("should rollback when github rename fails", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))

		renameCount := 0
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"remote", "get-url", "github"}, "git@github.com:owner/repo-a.git")
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) >= 4 && args[0] == "remote" && args[1] == "rename" {
				renameCount++
				if renameCount == 2 {
					return errors.New("rename failed")
				}
			}
			return nil
		}
		var buf bytes.Buffer

		// when
		err := repo.RunRestore(repo.RestoreConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "FAIL (rename github")
	})
}
