package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildGvmUseCommand(t *testing.T) {
	t.Parallel()

	t.Run("should resolve highest installed patch when version is 2-segment", func(t *testing.T) {
		t.Parallel()
		// given
		version := "1.26"

		// when
		cmd := buildGvmUseCommand(version)

		// then
		assert.Contains(t, cmd, `gvm list 2>/dev/null`)
		assert.Contains(t, cmd, `grep -E '^go1\.26(\.[0-9]+)?$'`)
		assert.Contains(t, cmd, `sort -V | tail -n1`)
		assert.Contains(t, cmd, `gvm use "$_dev_go"`)
		assert.Contains(t, cmd, `[dev] no installed Go matching 1.26`)
		assert.Contains(t, cmd, `gvm install go1.26.<patch>`)
	})

	t.Run("should guard exact tag when version is 3-segment", func(t *testing.T) {
		t.Parallel()
		// given
		version := "1.26.1"

		// when
		cmd := buildGvmUseCommand(version)

		// then
		assert.Contains(t, cmd, `grep -qxF go1.26.1`)
		assert.Contains(t, cmd, `gvm use go1.26.1`)
		assert.Contains(t, cmd, `[dev] go1.26.1 is not installed`)
		assert.Contains(t, cmd, `gvm install go1.26.1`)
	})

	t.Run("should produce shell that avoids calling gvm use when nothing matches", func(t *testing.T) {
		t.Parallel()
		// given
		version := "1.99"

		// when
		cmd := buildGvmUseCommand(version)

		// then
		// The "else" branch must emit the hint and NOT contain a standalone `gvm use`
		// call -- this is the bug the wrapper prevents.
		assert.Contains(t, cmd, `else echo "[dev] no installed Go matching 1.99`)
		assert.NotContains(t, cmd, `else gvm use`)
	})
}

func TestBuildUseCommandDispatchesByManager(t *testing.T) {
	t.Parallel()

	t.Run("should emit pyenv local for pyenv", func(t *testing.T) {
		t.Parallel()
		// given / when
		cmd := buildUseCommand("pyenv", "python", "3.12.0")

		// then
		assert.Equal(t, "pyenv local 3.12.0", cmd)
	})

	t.Run("should emit nvm use for nvm", func(t *testing.T) {
		t.Parallel()
		// given / when
		cmd := buildUseCommand("nvm", "node", "20.11.0")

		// then
		assert.Equal(t, "nvm use 20.11.0", cmd)
	})

	t.Run("should emit sdk use java for sdkman", func(t *testing.T) {
		t.Parallel()
		// given / when
		cmd := buildUseCommand("sdkman", "java", "21-tem")

		// then
		assert.Equal(t, "sdk use java 21-tem", cmd)
	})

	t.Run("should delegate to gvm builder for gvm", func(t *testing.T) {
		t.Parallel()
		// given / when
		cmd := buildUseCommand("gvm", "go", "1.26.0")

		// then
		assert.Contains(t, cmd, `grep -qxF go1.26.0`)
		assert.Contains(t, cmd, `gvm use go1.26.0`)
	})

	t.Run("should return empty for unknown manager", func(t *testing.T) {
		t.Parallel()
		// given / when
		cmd := buildUseCommand("unknown", "go", "1.26.0")

		// then
		assert.Empty(t, cmd)
	})
}
