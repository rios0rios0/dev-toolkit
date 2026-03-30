package system_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/system"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunTop5Size(t *testing.T) {
	t.Parallel()

	t.Run("should display top 5 items sorted by size descending", func(t *testing.T) {
		t.Parallel()
		// given
		entries := []os.DirEntry{
			&doubles.FakeDirEntry{EntryName: "small"},
			&doubles.FakeDirEntry{EntryName: "medium"},
			&doubles.FakeDirEntry{EntryName: "large"},
			&doubles.FakeDirEntry{EntryName: "huge"},
			&doubles.FakeDirEntry{EntryName: "tiny"},
			&doubles.FakeDirEntry{EntryName: "extra"},
		}
		fs := doubles.NewFileSystemStub().WithReadDir("/test", entries)
		runner := doubles.NewSystemRunnerStub()
		runner.OutputFunc = func(_ string, _ ...string) (string, error) {
			return "1024\t/test/small\n" +
				"1048576\t/test/medium\n" +
				"1073741824\t/test/large\n" +
				"5368709120\t/test/huge\n" +
				"512\t/test/tiny\n" +
				"2048\t/test/extra", nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunTop5Size(runner, fs, "/test", false, &buf)

		// then
		require.NoError(t, err)
		lines := nonEmptyLines(buf.String())
		require.Len(t, lines, 5)
		assert.Contains(t, lines[0], "huge")
		assert.Contains(t, lines[1], "large")
		assert.Contains(t, lines[2], "medium")
	})

	t.Run("should handle directory with fewer than 5 items", func(t *testing.T) {
		t.Parallel()
		// given
		entries := []os.DirEntry{
			&doubles.FakeDirEntry{EntryName: "one"},
			&doubles.FakeDirEntry{EntryName: "two"},
		}
		fs := doubles.NewFileSystemStub().WithReadDir("/test", entries)
		runner := doubles.NewSystemRunnerStub()
		runner.OutputFunc = func(_ string, _ ...string) (string, error) {
			return "2048\t/test/one\n1024\t/test/two", nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunTop5Size(runner, fs, "/test", false, &buf)

		// then
		require.NoError(t, err)
		lines := nonEmptyLines(buf.String())
		assert.Len(t, lines, 2)
		assert.Contains(t, lines[0], "one")
		assert.Contains(t, lines[1], "two")
	})

	t.Run("should handle empty directory", func(t *testing.T) {
		t.Parallel()
		// given
		fs := doubles.NewFileSystemStub().WithReadDir("/test", []os.DirEntry{})
		runner := doubles.NewSystemRunnerStub()
		var buf bytes.Buffer

		// when
		err := system.RunTop5Size(runner, fs, "/test", false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "directory is empty")
	})

	t.Run("should return error when reading directory fails", func(t *testing.T) {
		t.Parallel()
		// given
		fs := doubles.NewFileSystemStub().WithReadDirError("/test", errors.New("permission denied"))
		runner := doubles.NewSystemRunnerStub()
		var buf bytes.Buffer

		// when
		err := system.RunTop5Size(runner, fs, "/test", false, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading directory")
	})

	t.Run("should return error when du command fails", func(t *testing.T) {
		t.Parallel()
		// given
		entries := []os.DirEntry{&doubles.FakeDirEntry{EntryName: "file"}}
		fs := doubles.NewFileSystemStub().WithReadDir("/test", entries)
		runner := doubles.NewSystemRunnerStub()
		runner.OutputFunc = func(_ string, _ ...string) (string, error) {
			return "", errors.New("du not found")
		}
		var buf bytes.Buffer

		// when
		err := system.RunTop5Size(runner, fs, "/test", false, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "running du")
	})

	t.Run("should prepend sudo when useSudo is true", func(t *testing.T) {
		t.Parallel()
		// given
		entries := []os.DirEntry{&doubles.FakeDirEntry{EntryName: "file"}}
		fs := doubles.NewFileSystemStub().WithReadDir("/test", entries)
		var capturedName string
		runner := doubles.NewSystemRunnerStub()
		runner.OutputFunc = func(name string, _ ...string) (string, error) {
			capturedName = name
			return "1024\t/test/file", nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunTop5Size(runner, fs, "/test", true, &buf)

		// then
		require.NoError(t, err)
		assert.Equal(t, "sudo", capturedName)
	})

	t.Run("should not prepend sudo when useSudo is false", func(t *testing.T) {
		t.Parallel()
		// given
		entries := []os.DirEntry{&doubles.FakeDirEntry{EntryName: "file"}}
		fs := doubles.NewFileSystemStub().WithReadDir("/test", entries)
		var capturedName string
		runner := doubles.NewSystemRunnerStub()
		runner.OutputFunc = func(name string, _ ...string) (string, error) {
			capturedName = name
			return "1024\t/test/file", nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunTop5Size(runner, fs, "/test", false, &buf)

		// then
		require.NoError(t, err)
		assert.Equal(t, "du", capturedName)
	})
}

func nonEmptyLines(s string) []string {
	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		if len(strings.TrimSpace(line)) > 0 {
			lines = append(lines, line)
		}
	}
	return lines
}
