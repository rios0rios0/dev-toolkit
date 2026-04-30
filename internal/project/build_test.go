package project_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/project"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestRunBuild(t *testing.T) {
	t.Parallel()

	t.Run("should run all build commands when language detected successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:      "go",
			SDKName:       "Go",
			BuildCommands: []string{"go build ./..."},
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &buf,
		}

		// when
		err := project.RunBuild(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go build ./..."}, runner.Calls)
		assert.Contains(t, buf.String(), "build completed successfully")
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
		err := project.RunBuild(cfg)

		// then
		require.Error(t, err)
	})

	t.Run("should return error when build commands list is empty", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:      "unknown",
			BuildCommands: nil,
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunBuild(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no build commands")
	})

	t.Run("should return error and stop when a build command fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub().WithError(errors.New("compilation failed"))
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:      "go",
			SDKName:       "Go",
			BuildCommands: []string{"go build ./...", "go vet ./..."},
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunBuild(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "build command")
		assert.Len(t, runner.Calls, 1)
	})

	t.Run("should run multiple build commands in order", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:      "go",
			SDKName:       "Go",
			BuildCommands: []string{"go generate ./...", "go build ./..."},
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunBuild(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go generate ./...", "go build ./..."}, runner.Calls)
	})
}
