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

func TestRunStartWithDeps(t *testing.T) {
	t.Parallel()

	t.Run("should start dependencies before the project", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorMultiStub().
			WithInfo("/projects/auth", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StartCommand: "go run ./cmd/auth",
			}).
			WithInfo("/projects/api", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StartCommand: "go run ./cmd/api",
			})
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath:     "/projects/api",
			Detector:     detector,
			Runner:       runner,
			ConfigReader: reader,
			Output:       &buf,
		}

		// when
		err := project.RunStartWithDeps(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go run ./cmd/auth", "go run ./cmd/api"}, runner.Calls)
		assert.Contains(t, buf.String(), "starting dependency: /projects/auth")
		assert.Contains(t, buf.String(), "starting project: /projects/api")
	})

	t.Run("should fall back to single-project start when ConfigReader is nil", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "go", SDKName: "Go", StartCommand: "go run .",
		})
		cfg := project.Config{
			RepoPath:     "/tmp/my-project",
			Detector:     detector,
			Runner:       runner,
			ConfigReader: nil,
			Output:       &bytes.Buffer{},
		}

		// when
		err := project.RunStartWithDeps(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go run ."}, runner.Calls)
	})

	t.Run("should fall back to single-project start when no .dev.yaml exists", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "go", SDKName: "Go", StartCommand: "go run .",
		})
		reader := doubles.NewConfigReaderStub()
		cfg := project.Config{
			RepoPath:     "/tmp/my-project",
			Detector:     detector,
			Runner:       runner,
			ConfigReader: reader,
			Output:       &bytes.Buffer{},
		}

		// when
		err := project.RunStartWithDeps(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"go run ."}, runner.Calls)
	})

	t.Run("should return error when dependency fails to start", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub().WithError(errors.New("port in use"))
		detector := doubles.NewLanguageDetectorMultiStub().
			WithInfo("/projects/auth", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StartCommand: "go run ./cmd/auth",
			}).
			WithInfo("/projects/api", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StartCommand: "go run ./cmd/api",
			})
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			})
		cfg := project.Config{
			RepoPath:     "/projects/api",
			Detector:     detector,
			Runner:       runner,
			ConfigReader: reader,
			Output:       &bytes.Buffer{},
		}

		// when
		err := project.RunStartWithDeps(cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start /projects/auth")
		assert.Contains(t, err.Error(), "port in use")
	})

	t.Run("should return error when cycle detected", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			}).
			WithConfig("/projects/auth", &project.DevConfig{
				Dependencies: []string{"../api"},
			})
		cfg := project.Config{
			RepoPath:     "/projects/api",
			Detector:     doubles.NewLanguageDetectorStub(nil),
			Runner:       doubles.NewCommandRunnerStub(),
			ConfigReader: reader,
			Output:       &bytes.Buffer{},
		}

		// when
		err := project.RunStartWithDeps(cfg)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, project.ErrDependencyCycle)
	})
}

func TestRunStopWithDeps(t *testing.T) {
	t.Parallel()

	t.Run("should stop project before dependencies", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorMultiStub().
			WithInfo("/projects/auth", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StopCommand: "pkill auth",
			}).
			WithInfo("/projects/api", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StopCommand: "pkill api",
			})
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			})
		var buf bytes.Buffer
		cfg := project.Config{
			RepoPath:     "/projects/api",
			Detector:     detector,
			Runner:       runner,
			ConfigReader: reader,
			Output:       &buf,
		}

		// when
		err := project.RunStopWithDeps(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"pkill api", "pkill auth"}, runner.Calls)
		assert.Contains(t, buf.String(), "stopping project: /projects/api")
		assert.Contains(t, buf.String(), "stopping dependency: /projects/auth")
	})

	t.Run("should continue stopping all dependencies when one fails", func(t *testing.T) {
		t.Parallel()
		// given
		callCount := 0
		runner := &doubles.CommandRunnerStub{
			RunInteractiveFunc: func(_, _ string) error {
				callCount++
				if callCount == 1 {
					return errors.New("stop failed")
				}
				return nil
			},
		}
		detector := doubles.NewLanguageDetectorMultiStub().
			WithInfo("/projects/auth", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StopCommand: "pkill auth",
			}).
			WithInfo("/projects/api", &project.LanguageInfo{
				Language: "go", SDKName: "Go", StopCommand: "pkill api",
			})
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			})
		cfg := project.Config{
			RepoPath:     "/projects/api",
			Detector:     detector,
			Runner:       runner,
			ConfigReader: reader,
			Output:       &bytes.Buffer{},
		}

		// when
		err := project.RunStopWithDeps(cfg)

		// then
		require.Error(t, err)
		assert.Len(t, runner.Calls, 2, "should attempt to stop both projects")
	})

	t.Run("should fall back to single-project stop when ConfigReader is nil", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCommandRunnerStub()
		detector := doubles.NewLanguageDetectorStub(&project.LanguageInfo{
			Language: "go", SDKName: "Go", StopCommand: "pkill app",
		})
		cfg := project.Config{
			RepoPath:     "/tmp/my-project",
			Detector:     detector,
			Runner:       runner,
			ConfigReader: nil,
			Output:       &bytes.Buffer{},
		}

		// when
		err := project.RunStopWithDeps(cfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"pkill app"}, runner.Calls)
	})
}
