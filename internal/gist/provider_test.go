package gist_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/gist"
	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestResolveProviderWith(t *testing.T) {
	t.Parallel()

	t.Run("should build a provider when the credential resolves", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub().
			WithCredential(repo.Credential{Token: "cli-token", Source: "gh CLI"})

		// when
		provider, err := gist.ResolveProviderWith(resolver)

		// then
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, repo.ProviderGitHub, resolver.LastProvider)
	})

	t.Run("should return an error when the credential does not resolve", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub().WithError(errors.New("not authenticated"))

		// when
		_, err := gist.ResolveProviderWith(resolver)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not authenticated")
	})
}
