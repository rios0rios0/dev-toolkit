package repo_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/repo"
	"github.com/rios0rios0/devforge/internal/testutil/builders"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestComputeDiff(t *testing.T) {
	t.Parallel()

	t.Run("should return no missing and no extra when repos match", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
			builders.NewRepositoryBuilder().WithName("repo-b").Build(),
		}
		local := []string{"repo-a", "repo-b"}

		// when
		missing, extra := repo.ComputeDiff(remote, local)

		// then
		assert.Empty(t, missing)
		assert.Empty(t, extra)
	})

	t.Run("should return missing repos when remote has repos not in local", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
			builders.NewRepositoryBuilder().WithName("repo-b").Build(),
			builders.NewRepositoryBuilder().WithName("repo-c").Build(),
		}
		local := []string{"repo-a"}

		// when
		missing, extra := repo.ComputeDiff(remote, local)

		// then
		assert.Len(t, missing, 2)
		assert.Empty(t, extra)
	})

	t.Run("should return extra repos when local has repos not in remote", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
		}
		local := []string{"repo-a", "repo-b", "repo-c"}

		// when
		missing, extra := repo.ComputeDiff(remote, local)

		// then
		assert.Empty(t, missing)
		assert.Len(t, extra, 2)
		assert.Contains(t, extra, "repo-b")
		assert.Contains(t, extra, "repo-c")
	})

	t.Run("should return both missing and extra when sets differ", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("remote-only").Build(),
		}
		local := []string{"local-only"}

		// when
		missing, extra := repo.ComputeDiff(remote, local)

		// then
		assert.Len(t, missing, 1)
		assert.Equal(t, "remote-only", missing[0].Name)
		assert.Len(t, extra, 1)
		assert.Contains(t, extra, "local-only")
	})

	t.Run("should handle empty remote list", func(t *testing.T) {
		t.Parallel()
		// given
		local := []string{"repo-a"}

		// when
		missing, extra := repo.ComputeDiff(nil, local)

		// then
		assert.Empty(t, missing)
		assert.Len(t, extra, 1)
	})

	t.Run("should handle empty local list", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
		}

		// when
		missing, extra := repo.ComputeDiff(remote, nil)

		// then
		assert.Len(t, missing, 1)
		assert.Empty(t, extra)
	})

	t.Run("should handle Azure DevOps repos with project prefix", func(t *testing.T) {
		t.Parallel()
		// given
		remote := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("catalog").WithProject("backend").Build(),
		}
		local := []string{"backend/catalog"}

		// when
		missing, extra := repo.ComputeDiff(remote, local)

		// then
		assert.Empty(t, missing)
		assert.Empty(t, extra)
	})
}

func TestDiscoverRepos(t *testing.T) {
	t.Parallel()

	t.Run("should filter archived repos when includeArchived is false", func(t *testing.T) {
		t.Parallel()
		// given
		provider := doubles.NewForgeProviderStub().WithRepos([]globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("active").WithArchived(false).Build(),
			builders.NewRepositoryBuilder().WithName("archived").WithArchived(true).Build(),
		})
		var buf bytes.Buffer

		// when
		repos, err := repo.DiscoverRepos(provider, "owner", false, &buf)

		// then
		require.NoError(t, err)
		assert.Len(t, repos, 1)
		assert.Equal(t, "active", repos[0].Name)
	})

	t.Run("should include archived repos when includeArchived is true", func(t *testing.T) {
		t.Parallel()
		// given
		provider := doubles.NewForgeProviderStub().WithRepos([]globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("active").WithArchived(false).Build(),
			builders.NewRepositoryBuilder().WithName("archived").WithArchived(true).Build(),
		})
		var buf bytes.Buffer

		// when
		repos, err := repo.DiscoverRepos(provider, "owner", true, &buf)

		// then
		require.NoError(t, err)
		assert.Len(t, repos, 2)
	})

	t.Run("should return error when provider discovery fails", func(t *testing.T) {
		t.Parallel()
		// given
		provider := doubles.NewForgeProviderStub().WithDiscoverError(errors.New("API error"))
		var buf bytes.Buffer

		// when
		_, err := repo.DiscoverRepos(provider, "owner", false, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to discover repositories")
	})
}

func TestCloneMissing(t *testing.T) {
	t.Parallel()

	t.Run("should return zero counts when no missing repos", func(t *testing.T) {
		t.Parallel()
		// given
		cfg := repo.CloneConfig{
			RootDir:  "/tmp/test",
			DryRun:   false,
			Output:   &bytes.Buffer{},
			Provider: doubles.NewForgeProviderStub(),
		}

		// when
		cloned, failed := repo.CloneMissing(nil, cfg)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 0, failed)
	})

	t.Run("should clone repos when not in dry-run mode", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		provider := doubles.NewForgeProviderStub()
		runner := doubles.NewGitRunnerStub()
		noopPreflight := func(_, _ string, _ io.Writer) error { return nil }
		missing := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
		}
		cfg := repo.CloneConfig{
			RootDir:   "/home/user/Development/github.com/owner",
			SSHAlias:  "mine",
			DryRun:    false,
			Output:    &buf,
			Provider:  provider,
			Runner:    runner,
			Preflight: noopPreflight,
		}

		// when
		cloned, failed := repo.CloneMissing(missing, cfg)

		// then
		assert.Equal(t, 1, cloned)
		assert.Equal(t, 0, failed)
		assert.Contains(t, buf.String(), "CLONED")
	})

	t.Run("should return all failed when preflight fails", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		provider := doubles.NewForgeProviderStub()
		failPreflight := func(_, _ string, _ io.Writer) error { return errors.New("ssh failed") }
		missing := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
			builders.NewRepositoryBuilder().WithName("repo-b").Build(),
		}
		cfg := repo.CloneConfig{
			RootDir:   "/home/user/Development/github.com/owner",
			SSHAlias:  "mine",
			DryRun:    false,
			Output:    &buf,
			Provider:  provider,
			Preflight: failPreflight,
		}

		// when
		cloned, failed := repo.CloneMissing(missing, cfg)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 2, failed)
		assert.Contains(t, buf.String(), "ERROR")
	})

	t.Run("should log URLs without cloning in dry-run mode", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		provider := doubles.NewForgeProviderStub()
		missing := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
		}
		cfg := repo.CloneConfig{
			RootDir:  "/home/user/Development/github.com/owner",
			SSHAlias: "mine",
			DryRun:   true,
			Output:   &buf,
			Provider: provider,
		}

		// when
		cloned, failed := repo.CloneMissing(missing, cfg)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 0, failed)
		assert.Contains(t, buf.String(), "would clone")
	})
}

func TestRunClone(t *testing.T) {
	t.Parallel()

	t.Run("should complete successfully with no missing and no extra", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/repo-a")
		provider := doubles.NewForgeProviderStub().WithRepos([]globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
		})
		var buf bytes.Buffer
		cfg := repo.CloneConfig{
			RootDir:  "/home/user/Development/github.com/owner",
			SSHAlias: "mine",
			DryRun:   true,
			Provider: provider,
			Runner:   doubles.NewGitRunnerStub(),
			Output:   &buf,
			Input:    strings.NewReader(""),
		}

		// when
		err := repo.RunClone(cfg)

		// then
		assert.NoError(t, err)
	})

	t.Run("should return error when provider cannot be detected", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		cfg := repo.CloneConfig{
			RootDir: "/invalid/path",
			Output:  &buf,
			Input:   strings.NewReader(""),
		}

		// when
		err := repo.RunClone(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not detect provider")
	})

	t.Run("should return error when discovery fails", func(t *testing.T) {
		t.Parallel()
		// given
		provider := doubles.NewForgeProviderStub().WithDiscoverError(errors.New("API down"))
		var buf bytes.Buffer
		cfg := repo.CloneConfig{
			RootDir:  "/home/user/Development/github.com/owner",
			SSHAlias: "mine",
			Provider: provider,
			Output:   &buf,
			Input:    strings.NewReader(""),
		}

		// when
		err := repo.RunClone(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to discover")
	})

	t.Run("should report everything in sync when no diff", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		provider := doubles.NewForgeProviderStub().WithRepos(nil)
		var buf bytes.Buffer
		cfg := repo.CloneConfig{
			RootDir:  "/home/user/Development/github.com/owner",
			SSHAlias: "mine",
			Provider: provider,
			Runner:   doubles.NewGitRunnerStub(),
			Output:   &buf,
			Input:    strings.NewReader(""),
		}
		_ = root

		// when
		err := repo.RunClone(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "everything is in sync")
	})
}

func TestParallelClone(t *testing.T) {
	t.Parallel()

	t.Run("should clone all repos and return success counts", func(t *testing.T) {
		t.Parallel()
		// given
		repos := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
			builders.NewRepositoryBuilder().WithName("repo-b").Build(),
		}
		provider := doubles.NewForgeProviderStub()
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		cloned, failed := repo.ParallelClone(repos, provider, "mine", "/tmp/root", runner, &buf)

		// then
		assert.Equal(t, 2, cloned)
		assert.Equal(t, 0, failed)
		assert.Contains(t, buf.String(), "CLONED")
	})

	t.Run("should count failures when clone errors occur", func(t *testing.T) {
		t.Parallel()
		// given
		repos := []globalEntities.Repository{
			builders.NewRepositoryBuilder().WithName("repo-a").Build(),
			builders.NewRepositoryBuilder().WithName("repo-b").Build(),
		}
		provider := doubles.NewForgeProviderStub()
		runner := doubles.NewGitRunnerStub().WithCloneError(errors.New("permission denied"))
		var buf bytes.Buffer

		// when
		cloned, failed := repo.ParallelClone(repos, provider, "mine", "/tmp/root", runner, &buf)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 2, failed)
		assert.Contains(t, buf.String(), "FAIL")
	})

	t.Run("should handle empty repo list", func(t *testing.T) {
		t.Parallel()
		// given
		provider := doubles.NewForgeProviderStub()
		runner := doubles.NewGitRunnerStub()
		var buf bytes.Buffer

		// when
		cloned, failed := repo.ParallelClone(nil, provider, "mine", "/tmp/root", runner, &buf)

		// then
		assert.Equal(t, 0, cloned)
		assert.Equal(t, 0, failed)
	})
}

func TestPromptDeleteExtra(t *testing.T) {
	t.Parallel()

	t.Run("should keep repo in non-interactive mode with nil input", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/extra-repo")
		var buf bytes.Buffer

		// when
		repo.HandleExtraRepos([]string{"extra-repo"}, root, false, nil, &buf)

		// then
		assert.Contains(t, buf.String(), "kept, non-interactive")
	})

	t.Run("should keep repo when user answers n via PromptDeleteExtra", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/extra-repo")
		var buf bytes.Buffer
		input := strings.NewReader("n\n")

		// when
		repo.PromptDeleteExtra("extra-repo", root, input, &buf)

		// then
		assert.Contains(t, buf.String(), "kept extra-repo")
	})

	t.Run("should delete repo when user answers y via PromptDeleteExtra", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/extra-repo")
		var buf bytes.Buffer
		input := strings.NewReader("y\n")

		// when
		repo.PromptDeleteExtra("extra-repo", root, input, &buf)

		// then
		assert.Contains(t, buf.String(), "deleted extra-repo")
	})

	t.Run("should keep repo when scanner returns empty input", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, root+"/extra-repo")
		var buf bytes.Buffer
		input := strings.NewReader("")

		// when
		repo.PromptDeleteExtra("extra-repo", root, input, &buf)

		// then
		assert.Contains(t, buf.String(), "kept extra-repo")
	})
}

func TestIsSSHSuccess(t *testing.T) {
	t.Parallel()

	t.Run("should return true for Azure DevOps success message", func(t *testing.T) {
		t.Parallel()
		// given
		stderr := "remote: Shell access is not supported.\nshell request failed on channel 0"

		// when
		result := repo.IsSSHSuccess(stderr)

		// then
		assert.True(t, result)
	})

	t.Run("should return true for GitHub success message", func(t *testing.T) {
		t.Parallel()
		// given
		stderr := "Hi user! You've successfully authenticated, but GitHub does not provide shell access."

		// when
		result := repo.IsSSHSuccess(stderr)

		// then
		assert.True(t, result)
	})

	t.Run("should return true for GitLab success message", func(t *testing.T) {
		t.Parallel()
		// given
		stderr := "Welcome to GitLab, @user!"

		// when
		result := repo.IsSSHSuccess(stderr)

		// then
		assert.True(t, result)
	})

	t.Run("should return false for permission denied", func(t *testing.T) {
		t.Parallel()
		// given
		stderr := "Permission denied (publickey)."

		// when
		result := repo.IsSSHSuccess(stderr)

		// then
		assert.False(t, result)
	})

	t.Run("should return false for connection refused", func(t *testing.T) {
		t.Parallel()
		// given
		stderr := "ssh: connect to host github.com port 22: Connection refused"

		// when
		result := repo.IsSSHSuccess(stderr)

		// then
		assert.False(t, result)
	})

	t.Run("should return false for empty stderr", func(t *testing.T) {
		t.Parallel()
		// given
		stderr := ""

		// when
		result := repo.IsSSHSuccess(stderr)

		// then
		assert.False(t, result)
	})
}

func TestMaxCloneArgs(t *testing.T) {
	t.Parallel()

	t.Run("should return 2", func(t *testing.T) {
		t.Parallel()
		// given / when
		result := repo.MaxCloneArgs()

		// then
		assert.Equal(t, 2, result)
	})
}

func TestIsTerminal(t *testing.T) {
	t.Parallel()

	t.Run("should return false for non-file reader", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		input := strings.NewReader("")

		// when
		repo.HandleExtraRepos([]string{"repo"}, "/tmp", false, input, &buf)

		// then
		assert.Contains(t, buf.String(), "kept, non-interactive")
	})

	t.Run("should return false for pipe file descriptor", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer
		r, w, _ := os.Pipe()
		defer r.Close()
		_, _ = w.WriteString("")
		w.Close()

		// when
		repo.HandleExtraRepos([]string{"repo"}, "/tmp", false, r, &buf)

		// then
		assert.Contains(t, buf.String(), "kept, non-interactive")
	})
}

func TestHandleExtraRepos(t *testing.T) {
	t.Parallel()

	t.Run("should log extra repos in dry-run mode", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer

		// when
		repo.HandleExtraRepos([]string{"extra-repo"}, "/tmp", true, strings.NewReader(""), &buf)

		// then
		assert.Contains(t, buf.String(), "extra: extra-repo")
	})

	t.Run("should skip deletion in non-interactive mode", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer

		// when
		repo.HandleExtraRepos([]string{"extra-repo"}, "/tmp", false, strings.NewReader(""), &buf)

		// then
		assert.Contains(t, buf.String(), "kept, non-interactive")
	})

	t.Run("should log extra in dry-run mode with multiple repos", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer

		// when
		repo.HandleExtraRepos([]string{"a", "b"}, "/tmp", true, strings.NewReader(""), &buf)

		// then
		assert.Contains(t, buf.String(), "extra: a")
		assert.Contains(t, buf.String(), "extra: b")
	})

	t.Run("should do nothing when extra list is empty", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer

		// when
		repo.HandleExtraRepos(nil, "/tmp", false, strings.NewReader(""), &buf)

		// then
		assert.Empty(t, buf.String())
	})
}
