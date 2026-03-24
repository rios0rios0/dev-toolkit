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

func TestRunIPs(t *testing.T) {
	t.Parallel()

	t.Run("should list container names and IPs when containers are running", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub().
			WithOutput([]string{"ps", "-q"}, "abc123\ndef456").
			WithOutput(
				[]string{
					"inspect",
					"--format",
					"{{ .Name }}: {{ range .NetworkSettings.Networks }}{{ .IPAddress }} {{ end }}",
					"abc123",
					"def456",
				},
				"/my-app: 172.17.0.2\n/my-db: 172.17.0.3",
			)
		var buf bytes.Buffer

		// when
		err := docker.RunIPs(runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "my-app: 172.17.0.2")
		assert.Contains(t, buf.String(), "my-db: 172.17.0.3")
	})

	t.Run("should report no containers when none are running", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub()
		var buf bytes.Buffer

		// when
		err := docker.RunIPs(runner, &buf)

		// then
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "no running containers")
	})

	t.Run("should return error when docker ps fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub().
			WithOutputError([]string{"ps", "-q"}, errors.New("docker not running"))
		var buf bytes.Buffer

		// when
		err := docker.RunIPs(runner, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing containers")
	})

	t.Run("should return error when docker inspect fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewDockerRunnerStub().
			WithOutput([]string{"ps", "-q"}, "abc123").
			WithOutputError(
				[]string{
					"inspect",
					"--format",
					"{{ .Name }}: {{ range .NetworkSettings.Networks }}{{ .IPAddress }} {{ end }}",
					"abc123",
				},
				errors.New("inspect failed"),
			)
		var buf bytes.Buffer

		// when
		err := docker.RunIPs(runner, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inspecting containers")
	})
}
