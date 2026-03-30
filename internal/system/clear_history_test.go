package system_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/system"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunClearHistory(t *testing.T) {
	t.Parallel()

	t.Run("should remove all history files and glob matches", func(t *testing.T) {
		t.Parallel()
		// given
		var removed []string
		fs := doubles.NewFileSystemStub().
			WithHomeDir("/home/user").
			WithGlob("/home/user/.zcompdump*", []string{
				"/home/user/.zcompdump-host-5.9",
				"/home/user/.zcompdump",
			})
		fs.RemoveFunc = func(path string) error {
			removed = append(removed, path)
			return nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunClearHistory(fs, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "history cleared")
		assert.Contains(t, removed, "/home/user/.bash_history")
		assert.Contains(t, removed, "/home/user/.python_history")
		assert.Contains(t, removed, "/home/user/.zcompdump-host-5.9")
	})

	t.Run("should report dry-run without removing files", func(t *testing.T) {
		t.Parallel()
		// given
		removeCalled := false
		fs := doubles.NewFileSystemStub().WithHomeDir("/home/user")
		fs.RemoveFunc = func(_ string) error {
			removeCalled = true
			return nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunClearHistory(fs, true, &buf)

		// then
		require.NoError(t, err)
		assert.False(t, removeCalled)
		assert.Contains(t, buf.String(), "dry-run")
		assert.Contains(t, buf.String(), "would remove")
	})

	t.Run("should return error when home directory cannot be determined", func(t *testing.T) {
		t.Parallel()
		// given
		fs := doubles.NewFileSystemStub().WithHomeDirError(errors.New("no home"))
		var buf bytes.Buffer

		// when
		err := system.RunClearHistory(fs, false, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting home directory")
	})

	t.Run("should continue when individual file removal fails", func(t *testing.T) {
		t.Parallel()
		// given
		fs := doubles.NewFileSystemStub().
			WithHomeDir("/home/user").
			WithRemoveError("/home/user/.bash_history", errors.New("permission denied"))
		var buf bytes.Buffer

		// when
		err := system.RunClearHistory(fs, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "warning: permission denied")
		assert.Contains(t, buf.String(), "history cleared")
	})

	t.Run("should continue when glob expansion fails", func(t *testing.T) {
		t.Parallel()
		// given
		fs := doubles.NewFileSystemStub().
			WithHomeDir("/home/user").
			WithGlobError("/home/user/.zcompdump*", errors.New("bad pattern"))
		var buf bytes.Buffer

		// when
		err := system.RunClearHistory(fs, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "warning: glob")
		assert.Contains(t, buf.String(), "history cleared")
	})
}
