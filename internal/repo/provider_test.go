package repo_test

import (
	"bytes"
	"testing"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
)

func TestDetectProviderAndOwner(t *testing.T) {
	t.Parallel()

	t.Run("should return github and owner when path contains github.com", func(t *testing.T) {
		t.Parallel()
		// given
		rootDir := "/home/user/Development/github.com/rios0rios0"

		// when
		providerName, owner, err := repo.DetectProviderAndOwner(rootDir)

		// then
		require.NoError(t, err)
		assert.Equal(t, "github", providerName)
		assert.Equal(t, "rios0rios0", owner)
	})

	t.Run("should return azuredevops and owner when path contains dev.azure.com", func(t *testing.T) {
		t.Parallel()
		// given
		rootDir := "/home/user/Development/dev.azure.com/ContosoSecurity"

		// when
		providerName, owner, err := repo.DetectProviderAndOwner(rootDir)

		// then
		require.NoError(t, err)
		assert.Equal(t, "azuredevops", providerName)
		assert.Equal(t, "ContosoSecurity", owner)
	})

	t.Run("should return gitlab and owner when path contains gitlab.com", func(t *testing.T) {
		t.Parallel()
		// given
		rootDir := "/home/user/Development/gitlab.com/mygroup"

		// when
		providerName, owner, err := repo.DetectProviderAndOwner(rootDir)

		// then
		require.NoError(t, err)
		assert.Equal(t, "gitlab", providerName)
		assert.Equal(t, "mygroup", owner)
	})

	t.Run("should return error when path has no known provider", func(t *testing.T) {
		t.Parallel()
		// given
		rootDir := "/home/user/Development/bitbucket.org/owner"

		// when
		_, _, err := repo.DetectProviderAndOwner(rootDir)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not detect provider")
	})

	t.Run("should return error when owner segment is empty", func(t *testing.T) {
		t.Parallel()
		// given
		rootDir := "/home/user/Development/github.com/"

		// when
		_, _, err := repo.DetectProviderAndOwner(rootDir)

		// then
		require.Error(t, err)
	})
}

func TestKey(t *testing.T) {
	t.Parallel()

	t.Run("should return name when project is empty", func(t *testing.T) {
		t.Parallel()
		// given
		r := globalEntities.Repository{Name: "dev-toolkit"}

		// when
		key := repo.Key(r)

		// then
		assert.Equal(t, "dev-toolkit", key)
	})

	t.Run("should return project/name when project is set", func(t *testing.T) {
		t.Parallel()
		// given
		r := globalEntities.Repository{Name: "catalog", Project: "backend"}

		// when
		key := repo.Key(r)

		// then
		assert.Equal(t, "backend/catalog", key)
	})
}

func TestProviderScanDepth(t *testing.T) {
	t.Parallel()

	t.Run("should return 1 for github", func(t *testing.T) {
		t.Parallel()
		// given / when
		depth := repo.ProviderScanDepth("github")

		// then
		assert.Equal(t, 1, depth)
	})

	t.Run("should return 2 for azuredevops", func(t *testing.T) {
		t.Parallel()
		// given / when
		depth := repo.ProviderScanDepth("azuredevops")

		// then
		assert.Equal(t, repo.ScanDepthNested, depth)
	})
}

func TestProviderTokenEnv(t *testing.T) {
	t.Parallel()

	t.Run("should return GH_TOKEN for github", func(t *testing.T) {
		t.Parallel()
		// given / when
		env := repo.ProviderTokenEnv("github")

		// then
		assert.Equal(t, "GH_TOKEN", env)
	})

	t.Run("should return AZURE_DEVOPS_EXT_PAT for azuredevops", func(t *testing.T) {
		t.Parallel()
		// given / when
		env := repo.ProviderTokenEnv("azuredevops")

		// then
		assert.Equal(t, "AZURE_DEVOPS_EXT_PAT", env)
	})
}

func TestProviderHost(t *testing.T) {
	t.Parallel()

	t.Run("should return github.com for github", func(t *testing.T) {
		t.Parallel()
		// given / when
		host := repo.ProviderHost("github")

		// then
		assert.Equal(t, "github.com", host)
	})

	t.Run("should return dev.azure.com for azuredevops", func(t *testing.T) {
		t.Parallel()
		// given / when
		host := repo.ProviderHost("azuredevops")

		// then
		assert.Equal(t, "dev.azure.com", host)
	})

	t.Run("should return empty string for unknown provider", func(t *testing.T) {
		t.Parallel()
		// given / when
		host := repo.ProviderHost("unknown")

		// then
		assert.Empty(t, host)
	})
}

func TestNewProviderRegistry(t *testing.T) {
	t.Parallel()

	t.Run("should create registry with all providers", func(t *testing.T) {
		t.Parallel()
		// given / when
		registry := repo.NewProviderRegistry()

		// then
		assert.NotNil(t, registry)
	})
}

func TestNewLogger(t *testing.T) {
	t.Parallel()

	t.Run("should create a logger that writes to the given writer", func(t *testing.T) {
		t.Parallel()
		// given
		var buf bytes.Buffer

		// when
		log := repo.NewLogger(&buf)
		log.Info("test message")

		// then
		assert.Contains(t, buf.String(), "test message")
		assert.Contains(t, buf.String(), "level=info")
	})
}
