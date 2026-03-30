package system_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/devforge/internal/system"
	"github.com/rios0rios0/devforge/internal/testutil/doubles"
)

func TestRunClearLogs(t *testing.T) {
	t.Parallel()

	t.Run("should call find with sudo to remove old log files", func(t *testing.T) {
		t.Parallel()
		// given
		var capturedName string
		var capturedArgs []string
		runner := doubles.NewSystemRunnerStub()
		runner.RunFunc = func(name string, args ...string) error {
			capturedName = name
			capturedArgs = args
			return nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunClearLogs(runner, false, &buf)

		// then
		require.NoError(t, err)
		assert.Equal(t, "sudo", capturedName)
		assert.Equal(t, "find", capturedArgs[0])
		assert.Contains(t, capturedArgs, "/var/log")
		assert.Contains(t, capturedArgs, "*.log")
		assert.Contains(t, capturedArgs, "+5")
		assert.Contains(t, buf.String(), "old log files cleared")
	})

	t.Run("should report dry-run without executing", func(t *testing.T) {
		t.Parallel()
		// given
		runCalled := false
		runner := doubles.NewSystemRunnerStub()
		runner.RunFunc = func(_ string, _ ...string) error {
			runCalled = true
			return nil
		}
		var buf bytes.Buffer

		// when
		err := system.RunClearLogs(runner, true, &buf)

		// then
		require.NoError(t, err)
		assert.False(t, runCalled)
		assert.Contains(t, buf.String(), "dry-run")
		assert.Contains(t, buf.String(), "would remove log files")
	})

	t.Run("should return error when find command fails", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewSystemRunnerStub()
		runner.RunFunc = func(_ string, _ ...string) error {
			return errors.New("command failed")
		}
		var buf bytes.Buffer

		// when
		err := system.RunClearLogs(runner, false, &buf)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "clearing logs")
	})
}
