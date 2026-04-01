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

func TestRunTest(t *testing.T) {
	t.Parallel()

	t.Run("should run test commands when language detected successfully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "go",
			SDKName:      "Go",
			TestCommands: []string{"go test -tags unit ./..."},
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &buf,
		}

		// when
		err := project.RunTest(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go test -tags unit ./..."}, runner.Calls)
		assert.Contains(t, buf.String(), "tests completed successfully")
	})

	t.Run("should prefer explicit TestCommands over mapper", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "go",
			SDKName:      "Go",
			TestCommands: []string{"go test -v ./..."},
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &buf,
		}

		// when
		err := project.RunTest(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go test -v ./..."}, runner.Calls)
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
		err := project.RunTest(cfg)

		// then
		require.Error(t, err)
	})

	t.Run("should return error when no test commands available", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "unknown",
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunTest(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no test commands")
	})

	t.Run("should return error and stop when a test command fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub().WithError(errors.New("tests failed"))
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "python",
			SDKName:      "Python",
			TestCommands: []string{"pdm run pytest", "pdm run coverage"},
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunTest(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "test command")
		assert.Len(t, runner.Calls, 1)
	})

	t.Run("should run multiple test commands in order", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:     "node",
			SDKName:      "Node.js",
			TestCommands: []string{"npm run lint", "npm test"},
		})
		cfg := project.Config{
			RepoPath: "/tmp/my-project",
			Detector: detector,
			Runner:   runner,
			Output:   &bytes.Buffer{},
		}

		// when
		err := project.RunTest(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"npm run lint", "npm test"}, runner.Calls)
	})
}
