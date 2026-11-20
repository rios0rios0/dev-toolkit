package repo

import (
	"slices"
	"testing"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	gitRegistry "github.com/rios0rios0/gitforge/v4/pkg/registry/infrastructure"
	gitforgeDoubles "github.com/rios0rios0/gitforge/v4/test/doubles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	seamFixtureToken        = "fixture-token-placeholder"
	seamFixtureOrganization = "example-org"
)

// replaceCredentialResolver puts resolver in place of gitforge's default chain until the test
// ends, so the Resolve functions without a resolver argument ask it instead of the real
// environment, gh or az.
func replaceCredentialResolver(t *testing.T, resolver credentialEntities.CredentialResolver) {
	t.Helper()
	previous := newCredentialResolver
	newCredentialResolver = func() credentialEntities.CredentialResolver { return resolver }
	t.Cleanup(func() { newCredentialResolver = previous })
}

// TestResolveProviderForOwner is deliberately not parallel: it replaces the package-level
// newCredentialResolver, which every Resolve function without a resolver argument reads.
//
//nolint:paralleltest // replaces the package-level newCredentialResolver seam
func TestResolveProviderForOwner(t *testing.T) {
	providerNames := NewProviderRegistry().Names()
	slices.Sort(providerNames)

	for _, providerName := range providerNames {
		t.Run(
			"should send the owner as the organization when resolving the "+providerName+" provider",
			func(t *testing.T) {
				// given
				resolver := &gitforgeDoubles.CredentialResolverStub{
					Credential: credentialEntities.Credential{Token: seamFixtureToken, Source: "stub"},
				}
				replaceCredentialResolver(t, resolver)

				// when
				provider, source, err := ResolveProviderForOwner(providerName, seamFixtureOrganization)

				// then
				require.NoError(t, err)
				assert.Equal(t, providerName, provider.Name())
				assert.Equal(t, "stub", source)
				requests := resolver.Requests()
				require.Len(t, requests, 1)
				assert.Equal(t, providerName, gitRegistry.ServiceTypeToProviderName(requests[0].ServiceType))
				assert.Equal(t, seamFixtureOrganization, requests[0].Organization)
			},
		)
	}
}
