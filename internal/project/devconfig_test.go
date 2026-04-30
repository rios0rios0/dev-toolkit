package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/project"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestFileConfigReader(t *testing.T) {
	t.Parallel()

	t.Run("should return ErrNoConfig when no .dev.yaml exists", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		reader := &project.FileConfigReader{}

		// when
		cfg, err := reader.Read(dir)

		// then
		require.ErrorIs(t, err, project.ErrNoConfig)
		assert.Nil(t, cfg)
	})

	t.Run("should parse .dev.yaml with single dependency", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		content := "dependencies:\n  - ../service-auth\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".dev.yaml"), []byte(content), 0o600))
		reader := &project.FileConfigReader{}

		// when
		cfg, err := reader.Read(dir)

		// then
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, []string{"../service-auth"}, cfg.Dependencies)
	})

	t.Run("should parse .dev.yaml with multiple dependencies", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		content := "dependencies:\n  - ../service-auth\n  - ../service-gateway\n  - ../../shared/lib\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".dev.yaml"), []byte(content), 0o600))
		reader := &project.FileConfigReader{}

		// when
		cfg, err := reader.Read(dir)

		// then
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, []string{"../service-auth", "../service-gateway", "../../shared/lib"}, cfg.Dependencies)
	})

	t.Run("should return error when YAML is invalid", func(t *testing.T) {
		t.Parallel()
		// given
		dir := t.TempDir()
		content := ":::invalid yaml{{{"
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".dev.yaml"), []byte(content), 0o600))
		reader := &project.FileConfigReader{}

		// when
		cfg, err := reader.Read(dir)

		// then
		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse .dev.yaml")
	})
}

func TestResolveDependencyOrder(t *testing.T) {
	t.Parallel()

	t.Run("should return single-element list when no config exists", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub()

		// when
		order, err := project.ResolveDependencyOrder("/projects/api", reader)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"/projects/api"}, order)
	})

	t.Run("should handle dependency with no .dev.yaml", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			})

		// when
		order, err := project.ResolveDependencyOrder("/projects/api", reader)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"/projects/auth", "/projects/api"}, order)
	})

	t.Run("should resolve linear dependency chain in correct order", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../gateway"},
			}).
			WithConfig("/projects/gateway", &project.DevConfig{
				Dependencies: []string{"../auth"},
			})

		// when
		order, err := project.ResolveDependencyOrder("/projects/api", reader)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"/projects/auth", "/projects/gateway", "/projects/api"}, order)
	})

	t.Run("should resolve diamond dependency without duplicates", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../svc-b", "../svc-c"},
			}).
			WithConfig("/projects/svc-b", &project.DevConfig{
				Dependencies: []string{"../shared"},
			}).
			WithConfig("/projects/svc-c", &project.DevConfig{
				Dependencies: []string{"../shared"},
			})

		// when
		order, err := project.ResolveDependencyOrder("/projects/api", reader)

		// then
		require.NoError(t, err)
		assert.Len(t, order, 4)
		// shared must come before both svc-b and svc-c
		sharedIdx := indexOf(order, "/projects/shared")
		svcBIdx := indexOf(order, "/projects/svc-b")
		svcCIdx := indexOf(order, "/projects/svc-c")
		apiIdx := indexOf(order, "/projects/api")
		assert.Less(t, sharedIdx, svcBIdx)
		assert.Less(t, sharedIdx, svcCIdx)
		assert.Less(t, svcBIdx, apiIdx)
		assert.Less(t, svcCIdx, apiIdx)
	})

	t.Run("should detect direct cycle and return error", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			}).
			WithConfig("/projects/auth", &project.DevConfig{
				Dependencies: []string{"../api"},
			})

		// when
		order, err := project.ResolveDependencyOrder("/projects/api", reader)

		// then
		require.Error(t, err)
		assert.Nil(t, order)
		assert.ErrorIs(t, err, project.ErrDependencyCycle)
	})

	t.Run("should detect indirect cycle and return error", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../gateway"},
			}).
			WithConfig("/projects/gateway", &project.DevConfig{
				Dependencies: []string{"../auth"},
			}).
			WithConfig("/projects/auth", &project.DevConfig{
				Dependencies: []string{"../api"},
			})

		// when
		order, err := project.ResolveDependencyOrder("/projects/api", reader)

		// then
		require.Error(t, err)
		assert.Nil(t, order)
		assert.ErrorIs(t, err, project.ErrDependencyCycle)
	})

	t.Run("should return error when config reader fails", func(t *testing.T) {
		t.Parallel()
		// given
		reader := doubles.NewConfigReaderStub().
			WithConfig("/projects/api", &project.DevConfig{
				Dependencies: []string{"../auth"},
			}).
			WithError(assert.AnError)

		// when
		order, err := project.ResolveDependencyOrder("/projects/api", reader)

		// then
		require.Error(t, err)
		assert.Nil(t, order)
	})
}

func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}
