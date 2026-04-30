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

func TestRunLint(t *testing.T) {
	t.Parallel()

	t.Run("should run all lint commands when language detected successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "go",
			SDKName:      "Go",
			LintCommands: []string{"golangci-lint run --fix ."},
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &buf,
		}

		// when
		err := project.RunLint(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"golangci-lint run --fix ."}, runner.Calls)
		assert.Contains(t, buf.String(), "lint completed successfully")
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
		err := project.RunLint(cfg)

		// then
		require.Error(t, err)
	})

	t.Run("should return error when lint commands list is empty", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "unknown",
			LintCommands: nil,
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunLint(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no lint commands")
	})

	t.Run("should return error and stop when a lint command fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub().WithError(errors.New("lint failed"))
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "go",
			SDKName:      "Go",
			LintCommands: []string{"golangci-lint run --fix .", "go vet ./..."},
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunLint(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lint command")
		assert.Len(t, runner.Calls, 1)
	})

	t.Run("should run multiple lint commands in order", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "python",
			SDKName:      "Python",
			LintCommands: []string{"isort .", "black .", "flake8 ."},
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunLint(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"isort .", "black .", "flake8 ."}, runner.Calls)
	})
}
