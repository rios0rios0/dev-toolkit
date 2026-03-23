package project_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/project"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunInfo(t *testing.T) {
	t.Parallel()

	t.Run("should display all detected language info", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:       "go",
			SDKName:        "Go",
			VersionManager: "gvm",
			CurrentVersion: "1.26.1",
			StartCommand:   "go run .",
			StopCommand:    "",
			LintCommands:   []string{"golangci-lint run ./..."},
			BuildCommands:  []string{"go build ./..."},
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &buf,
		}

		// when
		err := project.RunInfo(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "language:        go")
		assert.Contains(t, buf.String(), "SDK:             Go")
		assert.Contains(t, buf.String(), "version manager: gvm")
		assert.Contains(t, buf.String(), "current version: 1.26.1")
		assert.Contains(t, buf.String(), "start command:   go run .")
		assert.Contains(t, buf.String(), "stop command:    (none)")
		assert.Contains(t, buf.String(), "build commands:  go build ./...")
	})

	t.Run("should display not-installed when current version is empty", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "python",
			SDKName:  "Python",
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &buf,
		}

		// when
		err := project.RunInfo(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "current version: (not installed)")
	})

	t.Run("should return error when language detection fails", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(nil).WithError(errors.New("no language"))
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunInfo(cfg)

		// then
		require.Error(t, err)
	})
}
