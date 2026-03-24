package docker_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/docker"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunReset(t *testing.T) {
	t.Parallel()

	t.Run("should stop containers and prune when containers exist", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub().
			WithOutput([]string{"container", "ls", "-aq"}, "abc123\ndef456")
		var buf bytes.Buffer

		// when
		err := docker.RunReset(runner, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "stopping all containers")
		assert.Contains(t, buf.String(), "pruning containers")
		assert.Contains(t, buf.String(), "pruning volumes")
		assert.Contains(t, buf.String(), "pruning networks")
		assert.Contains(t, buf.String(), "pruning build cache")
		assert.Contains(t, buf.String(), "reset complete")
	})

	t.Run("should report dry-run mode without executing", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub().
			WithOutput([]string{"container", "ls", "-aq"}, "abc123")
		var buf bytes.Buffer

		// when
		err := docker.RunReset(runner, true, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "dry-run")
		assert.Contains(t, buf.String(), "would stop all containers")
		assert.Contains(t, buf.String(), "would prune containers")
	})

	t.Run("should handle no running containers gracefully", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub()
		var buf bytes.Buffer

		// when
		err := docker.RunReset(runner, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no containers to stop")
		assert.Contains(t, buf.String(), "reset complete")
	})

	t.Run("should return error when listing containers fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub().
			WithOutputError([]string{"container", "ls", "-aq"}, errors.New("docker not running"))
		var buf bytes.Buffer

		// when
		err := docker.RunReset(runner, false, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing containers")
	})

	t.Run("should continue pruning when stop fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub().
			WithOutput([]string{"container", "ls", "-aq"}, "abc123").
			WithRunError([]string{"container", "stop", "-t", "5", "abc123"}, errors.New("stop failed"))
		var buf bytes.Buffer

		// when
		err := docker.RunReset(runner, false, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "could not be stopped")
		assert.Contains(t, buf.String(), "reset complete")
	})
}
