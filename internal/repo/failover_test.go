package repo_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/repo"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunFailover(t *testing.T) {
	t.Parallel()

	t.Run("should report no repos when directory is empty", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunFailover(repo.FailoverConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no git repositories found")
	})

	t.Run("should switch repos with codeberg remote", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))

		var renamedPairs [][]string
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"remote", "get-url", "github"}, "").
			WithOutput([]string{"remote", "get-url", "codeberg"}, "git@codeberg.org:owner/repo-a.git")
		runner.RunFunc = func(_ string, args ...string) error {
			if len(args) >= 4 && args[0] == "remote" && args[1] == "rename" {
				renamedPairs = append(renamedPairs, []string{args[2], args[3]})
			}
			return nil
		}
		var buf bytes.Buffer

		// when
		err := repo.RunFailover(repo.FailoverConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "switched")
		assert.Equal(t, [][]string{{"origin", "github"}, {"codeberg", "origin"}}, renamedPairs)
	})

	t.Run("should skip repos without codeberg remote", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		err := repo.RunFailover(repo.FailoverConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no codeberg remote")
	})

	t.Run("should skip repos already in failover state", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"remote", "get-url", "github"}, "git@github.com:owner/repo-a.git").
			WithOutput([]string{"remote", "get-url", "codeberg"}, "git@codeberg.org:owner/repo-a.git")
		var buf bytes.Buffer

		// when
		err := repo.RunFailover(repo.FailoverConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "already failed over")
	})

	t.Run("should rollback when codeberg rename fails", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))

		renameCount := 0
		runner := doubles.NewGitRunnerStub().
			WithOutput([]string{"remote", "get-url", "github"}, "").
			WithOutput([]string{"remote", "get-url", "codeberg"}, "git@codeberg.org:owner/repo-a.git")
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
		err := repo.RunFailover(repo.FailoverConfig{
			RootDir: root,
			Runner:  runner,
			Output:  repo.NewLogger(&buf),
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "FAIL (rename codeberg:")
	})
}
