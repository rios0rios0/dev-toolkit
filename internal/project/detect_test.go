package project_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/project"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunInfoDisplaysNoneForEmptyFields(t *testing.T) {
	t.Parallel()

	t.Run("should display none for empty optional fields", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "unknown",
		})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath: "/tmp/test",
			Detector: detector,
			Output:   &buf,
		}

		// when
		err := project.RunInfo(cfg)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "(none)")
		assert.Contains(t, buf.String(), "(not installed)")
	})
}

func TestRunStartUsesCurrentDirWhenPathEmpty(t *testing.T) {
	t.Parallel()

	t.Run("should resolve to current directory when path is empty", func(t *testing.T) {
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
