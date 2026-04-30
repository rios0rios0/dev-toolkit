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

func TestRunUse(t *testing.T) {
	t.Parallel()

	t.Run("should print use command when required version differs from current", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:        "go",
			SDKName:         "Go",
			VersionManager:  "gvm",
			CurrentVersion:  "1.25.0",
			RequiredVersion: "1.26.0",
			UseCommand:      "gvm use go1.26.0",
		})
		var stderr, stdout bytes.Buffer

		// when
		err := project.RunUse(project.Config{
			RepoPath: "/some/path",
			Detector: detector,
			Output:   &stderr,
			Stdout:   &stdout,
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "switching")
		assert.Equal(t, "gvm use go1.26.0\n", stdout.String())
	})

	t.Run("should print install and use commands when version is not installed", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:        "node",
			SDKName:         "Node.js",
			VersionManager:  "nvm",
			CurrentVersion:  "",
			RequiredVersion: "20.11.0",
			InstallCommand:  "nvm install 20.11.0",
			UseCommand:      "nvm use 20.11.0",
		})
		var stderr, stdout bytes.Buffer

		// when
		err := project.RunUse(project.Config{
			RepoPath: "/some/path",
			Detector: detector,
			Output:   &stderr,
			Stdout:   &stdout,
		})

		// then
		require.NoError(t, err)
		assert.Equal(t, "nvm install 20.11.0\nnvm use 20.11.0\n", stdout.String())
	})

	t.Run("should print nothing when required version matches current", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:        "python",
			SDKName:         "Python",
			VersionManager:  "pyenv",
			CurrentVersion:  "3.12.0",
			RequiredVersion: "3.12.0",
		})
		var stderr, stdout bytes.Buffer

		// when
		err := project.RunUse(project.Config{
			RepoPath: "/some/path",
			Detector: detector,
			Output:   &stderr,
			Stdout:   &stdout,
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "already active")
		assert.Empty(t, stdout.String())
	})

	t.Run("should report no version constraint when project has none", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language:       "go",
			SDKName:        "Go",
			VersionManager: "gvm",
		})
		var stderr, stdout bytes.Buffer

		// when
		err := project.RunUse(project.Config{
			RepoPath: "/some/path",
			Detector: detector,
			Output:   &stderr,
			Stdout:   &stdout,
		})

		// then
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "no version constraint")
		assert.Empty(t, stdout.String())
	})

	t.Run("should return error when language detection fails", func(t *testing.T) {
		t.Parallel()
		// given
		detector := doubles.NewLanguageDetectorStub(nil).WithError(errors.New("unknown language"))
		var stderr, stdout bytes.Buffer

		// when
		err := project.RunUse(project.Config{
			RepoPath: "/some/path",
			Detector: detector,
			Output:   &stderr,
			Stdout:   &stdout,
		})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown language")
	})
}
