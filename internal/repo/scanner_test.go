package repo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rios0rios0/devforge/internal/repo"
)

func createGitRepo(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(filepath.Join(path, ".git"), 0o750)
	assert.NoError(t, err)
}

func TestScanFlatRepos(t *testing.T) {
	t.Parallel()

	t.Run("should return repos with .git directories", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))
		createGitRepo(t, filepath.Join(root, "repo-b"))

		// when
		repos := repo.ScanFlatRepos(root)

		// then
		assert.Len(t, repos, 2)
		assert.Contains(t, repos, "repo-a")
		assert.Contains(t, repos, "repo-b")
	})

	t.Run("should skip directories without .git", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))
		_ = os.MkdirAll(filepath.Join(root, "not-a-repo"), 0o750)

		// when
		repos := repo.ScanFlatRepos(root)

		// then
		assert.Len(t, repos, 1)
		assert.Contains(t, repos, "repo-a")
	})

	t.Run("should skip non-directory entries", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))
		_ = os.WriteFile(filepath.Join(root, "file.txt"), []byte("hello"), 0o644)

		// when
		repos := repo.ScanFlatRepos(root)

		// then
		assert.Len(t, repos, 1)
	})

	t.Run("should return empty slice for empty directory", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()

		// when
		repos := repo.ScanFlatRepos(root)

		// then
		assert.Empty(t, repos)
	})

	t.Run("should return empty slice for nonexistent directory", func(t *testing.T) {
		t.Parallel()
		// given / when
		repos := repo.ScanFlatRepos("/nonexistent/path")

		// then
		assert.Empty(t, repos)
	})
}

func TestScanNestedRepos(t *testing.T) {
	t.Parallel()

	t.Run("should return project/repo paths for nested structure", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "backend", "catalog"))
		createGitRepo(t, filepath.Join(root, "backend", "auth"))
		createGitRepo(t, filepath.Join(root, "frontend", "app"))

		// when
		repos := repo.ScanNestedRepos(root)

		// then
		assert.Len(t, repos, 3)
		assert.Contains(t, repos, "backend/catalog")
		assert.Contains(t, repos, "backend/auth")
		assert.Contains(t, repos, "frontend/app")
	})

	t.Run("should skip empty project directories", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "backend", "catalog"))
		_ = os.MkdirAll(filepath.Join(root, "empty-project"), 0o750)

		// when
		repos := repo.ScanNestedRepos(root)

		// then
		assert.Len(t, repos, 1)
		assert.Contains(t, repos, "backend/catalog")
	})

	t.Run("should return empty slice for empty directory", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()

		// when
		repos := repo.ScanNestedRepos(root)

		// then
		assert.Empty(t, repos)
	})
}

func TestScanLocalRepos(t *testing.T) {
	t.Parallel()

	t.Run("should delegate to flat scan for depth 1", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "repo-a"))

		// when
		repos := repo.ScanLocalRepos(root, 1)

		// then
		assert.Len(t, repos, 1)
		assert.Contains(t, repos, "repo-a")
	})

	t.Run("should delegate to nested scan for depth 2", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "project", "repo-a"))

		// when
		repos := repo.ScanLocalRepos(root, repo.ScanDepthNested)

		// then
		assert.Len(t, repos, 1)
		assert.Contains(t, repos, "project/repo-a")
	})
}

func TestFindAllRepos(t *testing.T) {
	t.Parallel()

	t.Run("should walk and return all repos recursively", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		createGitRepo(t, filepath.Join(root, "a"))
		createGitRepo(t, filepath.Join(root, "nested", "b"))

		// when
		repos := repo.FindAllRepos(root)

		// then
		assert.Len(t, repos, 2)
	})

	t.Run("should skip the root directory itself", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		_ = os.MkdirAll(filepath.Join(root, ".git"), 0o750)
		createGitRepo(t, filepath.Join(root, "child"))

		// when
		repos := repo.FindAllRepos(root)

		// then
		assert.Len(t, repos, 1)
	})

	t.Run("should return empty slice when no repos found", func(t *testing.T) {
		t.Parallel()
		// given
		root := t.TempDir()
		_ = os.MkdirAll(filepath.Join(root, "empty"), 0o750)

		// when
		repos := repo.FindAllRepos(root)

		// then
		assert.Empty(t, repos)
	})
}
