package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rios0rios0/dev-toolkit/internal/project"
)

func TestReadRequiredSDKVersion(t *testing.T) {
	t.Parallel()

	t.Run("should extract Go SDK version from go.mod", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.26.1\n"), 0o644)

		// when
		version := project.ReadRequiredSDKVersion(dir, "go")

		// then
		assert.Equal(t, "1.26.1", version)
	})

	t.Run("should extract Go SDK version without patch", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.26\n"), 0o644)

		// when
		version := project.ReadRequiredSDKVersion(dir, "go")

		// then
		assert.Equal(t, "1.26", version)
	})

	t.Run("should extract Node version from .nvmrc", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("20.11.0\n"), 0o644)

		// when
		version := project.ReadRequiredSDKVersion(dir, "node")

		// then
		assert.Equal(t, "20.11.0", version)
	})

	t.Run("should extract Python version from pyproject.toml requires-python", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		content := `[project]
name = "my-project"
requires-python = ">=3.12.0"
`
		os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0o644)

		// when
		version := project.ReadRequiredSDKVersion(dir, "python")

		// then
		assert.Equal(t, "3.12.0", version)
	})

	t.Run("should return empty for unsupported language", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()

		// when
		version := project.ReadRequiredSDKVersion(dir, "rust")

		// then
		assert.Empty(t, version)
	})

	t.Run("should return empty when version file does not exist", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()

		// when
		version := project.ReadRequiredSDKVersion(dir, "go")

		// then
		assert.Empty(t, version)
	})
}
