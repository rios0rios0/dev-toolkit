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

func TestRunStart(t *testing.T) {
	t.Parallel()

	t.Run("should run start command when language detected successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "go",
			SDKName:      "Go",
			StartCommand: "go run .",
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &buf,
		}

		// when
		err := project.RunStart(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go run ."}, runner.Calls)
		assert.Contains(t, buf.String(), "detected Go project")
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
		err := project.RunStart(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no language")
	})

	t.Run("should return error when start command is empty", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "terraform",
			SDKName:      "Terraform",
			StartCommand: "",
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunStart(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no start command")
	})

	t.Run("should return error when command execution fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub().WithError(errors.New("exit status 1"))
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "go",
			SDKName:      "Go",
			StartCommand: "go run .",
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunStart(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exit status 1")
	})

	t.Run("should use current directory when path is empty", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "go",
			SDKName:      "Go",
			StartCommand: "go run .",
		})
		cfg := project.Config{
			RepoPath: "",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunStart(cfg)

		// then
		require.NoError(t, err)
		assert.Len(t, runner.Calls, 1)
	})
}
