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

func TestRunStop(t *testing.T) {
	t.Parallel()

	t.Run("should run stop command when language detected successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:    "node",
			SDKName:     "Node.js",
			StopCommand: "npm stop",
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &buf,
		}

		// when
		err := project.RunStop(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"npm stop"}, runner.Calls)
		assert.Contains(t, buf.String(), "detected Node.js project")
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
		err := project.RunStop(cfg)

		// then
		require.Error(t, err)
	})

	t.Run("should return error when stop command is empty", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "go",
			SDKName:  "Go",
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunStop(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no stop command")
	})

	t.Run("should return error when command execution fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub().WithError(errors.New("process not found"))
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:    "node",
			SDKName:     "Node.js",
			StopCommand: "npm stop",
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunStop(cfg)

		// then
		require.Error(t, err)
	})
}
