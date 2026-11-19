package gist_test

import (
	"errors"
	"testing"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
	gitforgeDoubles "github.com/rios0rios0/gitforge/v4/test/doubles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/gist"
)

func TestResolveProviderWith(t *testing.T) {
	t.Parallel()

	t.Run("should build a provider from the GitHub credential when it resolves", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &gitforgeDoubles.CredentialResolverStub{
			Credential: credentialEntities.Credential{Token: "fixture-token-placeholder", Source: "gh CLI"},
		}

		// when
		provider, err := gist.ResolveProviderWith(resolver)

		// then
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, []credentialEntities.CredentialRequest{
			{ServiceType: globalEntities.GITHUB},
		}, resolver.Requests())
	})

	t.Run("should return the resolver's error when the credential does not resolve", func(t *testing.T) {
		t.Parallel()
		// given
		notAuthenticated := errors.New("not authenticated")
		resolver := &gitforgeDoubles.CredentialResolverStub{Err: notAuthenticated}

		// when
		_, err := gist.ResolveProviderWith(resolver)

		// then
		require.ErrorIs(t, err, notAuthenticated)
	})

	t.Run("should return an error when the GitHub client cannot be built", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &gitforgeDoubles.CredentialResolverStub{
			Credential: credentialEntities.Credential{Token: "", Source: "stub"},
		}

		// when
		provider, err := gist.ResolveProviderWith(resolver)

		// then
		require.ErrorContains(t, err, "failed to create GitHub client")
		assert.Nil(t, provider)
	})
}
