package repo_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	credentialInfra "github.com/rios0rios0/gitforge/v4/pkg/credentials/infrastructure"
	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
	gitRegistry "github.com/rios0rios0/gitforge/v4/pkg/registry/infrastructure"
	gitforgeDoubles "github.com/rios0rios0/gitforge/v4/test/doubles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
)

const (
	fixtureToken        = "fixture-token-placeholder"
	fixtureOtherToken   = "fixture-other-token-placeholder"
	fixturePAT          = "fixture-pat-placeholder"
	fixtureOrganization = "example-org"
	fixtureTenant       = "11111111-2222-3333-4444-555555555555"
)

// stubCredential returns a resolver that hands out the fixture token for every request.
func stubCredential() *gitforgeDoubles.CredentialResolverStub {
	return &gitforgeDoubles.CredentialResolverStub{
		Credential: credentialEntities.Credential{Token: fixtureToken, Source: "stub"},
	}
}

// newCredentialChain builds gitforge's chain in its default order -- environment, then
// CLI -- over an environment, CLIs and a tenant lookup that exist only in the test.
func newCredentialChain(
	env map[string]string,
	runner *gitforgeDoubles.CLIRunnerStub,
	tenants *gitforgeDoubles.AzureTenantResolverStub,
) *credentialInfra.ChainCredentialResolver {
	return credentialInfra.NewChainCredentialResolver(
		&credentialInfra.EnvCredentialResolver{Lookup: func(name string) string { return env[name] }},
		&credentialInfra.CLICredentialResolver{Runner: runner, TenantResolver: tenants},
	)
}

// registeredProviders returns every provider the registry builds, sorted so the subtests run
// in a stable order. A provider registered there without a ServiceType of its own fails the
// tests that loop over them.
func registeredProviders() []string {
	providerNames := repo.NewProviderRegistry().Names()
	slices.Sort(providerNames)
	return providerNames
}

func TestResolveProviderWith(t *testing.T) {
	t.Parallel()

	for _, providerName := range registeredProviders() {
		t.Run("should request the "+providerName+" credential when building its provider", func(t *testing.T) {
			t.Parallel()
			// given
			resolver := stubCredential()

			// when
			provider, source, err := repo.ResolveProviderWith(providerName, "", resolver)

			// then
			require.NoError(t, err)
			assert.Equal(t, providerName, provider.Name())
			assert.Equal(t, fixtureToken, provider.AuthToken())
			assert.Equal(t, "stub", source)
			requests := resolver.Requests()
			require.Len(t, requests, 1)
			assert.Equal(t, providerName, gitRegistry.ServiceTypeToProviderName(requests[0].ServiceType))
		})
	}

	t.Run("should send the owner as the organization when an owner is given", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := stubCredential()

		// when
		_, _, err := repo.ResolveProviderWith(repo.ProviderAzureDevOps, fixtureOrganization, resolver)

		// then
		require.NoError(t, err)
		assert.Equal(t, []credentialEntities.CredentialRequest{
			{ServiceType: globalEntities.AZUREDEVOPS, Organization: fixtureOrganization},
		}, resolver.Requests())
	})

	t.Run("should return the resolver's error when no credential is available", func(t *testing.T) {
		t.Parallel()
		// given
		notAuthenticated := errors.New("not authenticated")
		resolver := &gitforgeDoubles.CredentialResolverStub{Err: notAuthenticated}

		// when
		_, _, err := repo.ResolveProviderWith(repo.ProviderGitHub, "", resolver)

		// then
		require.ErrorIs(t, err, notAuthenticated)
	})

	t.Run("should refuse an unknown provider without asking for a credential", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := stubCredential()

		// when
		_, _, err := repo.ResolveProviderWith("bitbucket", "", resolver)

		// then
		require.EqualError(t, err, "unknown provider: bitbucket")
		assert.Empty(t, resolver.Requests())
	})
}

func TestResolveForkSyncWith(t *testing.T) {
	t.Parallel()

	t.Run("should build the provider and the fork resolver from one credential when the provider is GitHub",
		func(t *testing.T) {
			t.Parallel()
			// given
			resolver := stubCredential()

			// when
			provider, forkResolver, source, err := repo.ResolveForkSyncWith(repo.ProviderGitHub, resolver)

			// then
			require.NoError(t, err)
			assert.Equal(t, repo.ProviderGitHub, provider.Name())
			assert.Equal(t, fixtureToken, provider.AuthToken())
			assert.NotNil(t, forkResolver)
			assert.Equal(t, "stub", source)
			assert.Equal(t, []credentialEntities.CredentialRequest{
				{ServiceType: globalEntities.GITHUB},
			}, resolver.Requests())
		})

	for _, providerName := range registeredProviders() {
		if providerName == repo.ProviderGitHub {
			continue
		}
		t.Run("should refuse "+providerName+" before asking for a credential when it has no fork resolution",
			func(t *testing.T) {
				t.Parallel()
				// given
				resolver := stubCredential()

				// when
				provider, forkResolver, _, err := repo.ResolveForkSyncWith(providerName, resolver)

				// then
				require.EqualError(t, err, "fork resolution not supported for provider: "+providerName)
				assert.Nil(t, provider)
				assert.Nil(t, forkResolver)
				assert.Empty(t, resolver.Requests())
			})
	}

	t.Run("should return the resolver's error when the credential does not resolve", func(t *testing.T) {
		t.Parallel()
		// given
		notAuthenticated := errors.New("not authenticated")
		resolver := &gitforgeDoubles.CredentialResolverStub{Err: notAuthenticated}

		// when
		_, _, _, err := repo.ResolveForkSyncWith(repo.ProviderGitHub, resolver)

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
		provider, forkResolver, _, err := repo.ResolveForkSyncWith(repo.ProviderGitHub, resolver)

		// then
		require.ErrorContains(t, err, "failed to create GitHub client")
		assert.Nil(t, provider)
		assert.Nil(t, forkResolver)
	})
}

func TestResolveProviderWithCredentialChain(t *testing.T) {
	t.Parallel()

	t.Run("should issue the az token in the organization's tenant when no PAT is exported", func(t *testing.T) {
		t.Parallel()
		// given
		runner := &gitforgeDoubles.CLIRunnerStub{Paths: map[string]string{"az": "/usr/bin/az"}, Token: fixtureToken}
		tenants := &gitforgeDoubles.AzureTenantResolverStub{Tenant: fixtureTenant}
		chain := newCredentialChain(nil, runner, tenants)

		// when
		provider, source, err := repo.ResolveProviderWith(repo.ProviderAzureDevOps, fixtureOrganization, chain)

		// then
		require.NoError(t, err)
		assert.Equal(t, fixtureToken, provider.AuthToken())
		assert.Equal(t, "az CLI", source)
		assert.Equal(t, []string{fixtureOrganization}, tenants.Organizations())
		calls := runner.Calls()
		require.Len(t, calls, 1)
		assert.Contains(t, strings.Join(calls[0], " "), "--tenant "+fixtureTenant)
	})

	t.Run("should use the exported PAT without discovering the tenant when one is exported", func(t *testing.T) {
		t.Parallel()
		// given
		runner := &gitforgeDoubles.CLIRunnerStub{Paths: map[string]string{"az": "/usr/bin/az"}, Token: fixtureToken}
		tenants := &gitforgeDoubles.AzureTenantResolverStub{Err: errors.New("metadata unavailable")}
		chain := newCredentialChain(map[string]string{"AZURE_DEVOPS_EXT_PAT": fixturePAT}, runner, tenants)

		// when
		provider, source, err := repo.ResolveProviderWith(repo.ProviderAzureDevOps, fixtureOrganization, chain)

		// then
		require.NoError(t, err)
		assert.Equal(t, fixturePAT, provider.AuthToken())
		assert.Equal(t, "AZURE_DEVOPS_EXT_PAT environment variable", source)
		assert.Empty(t, tenants.Organizations())
		assert.Empty(t, runner.Calls())
	})

	t.Run("should not scope the gh token to the owner when the provider is GitHub", func(t *testing.T) {
		t.Parallel()
		// given
		runner := &gitforgeDoubles.CLIRunnerStub{Paths: map[string]string{"gh": "/usr/bin/gh"}, Token: fixtureToken}
		tenants := &gitforgeDoubles.AzureTenantResolverStub{Tenant: fixtureTenant}
		chain := newCredentialChain(nil, runner, tenants)

		// when
		provider, _, err := repo.ResolveProviderWith(repo.ProviderGitHub, fixtureOrganization, chain)

		// then
		require.NoError(t, err)
		assert.Equal(t, fixtureToken, provider.AuthToken())
		assert.Empty(t, tenants.Organizations())
		calls := runner.Calls()
		require.Len(t, calls, 1)
		assert.NotContains(t, calls[0], "--tenant")
	})

	t.Run("should name the GitLab variables without running glab when no GitLab token is exported", func(t *testing.T) {
		t.Parallel()
		// given
		runner := &gitforgeDoubles.CLIRunnerStub{Paths: map[string]string{"glab": "/usr/bin/glab"}, Token: fixtureToken}
		chain := newCredentialChain(nil, runner, &gitforgeDoubles.AzureTenantResolverStub{})

		// when
		_, _, err := repo.ResolveProviderWith(repo.ProviderGitLab, "", chain)

		// then
		require.ErrorIs(t, err, credentialEntities.ErrNoCredential)
		require.ErrorContains(t, err, "GITLAB_TOKEN")
		require.ErrorContains(t, err, "GL_TOKEN")
		assert.Empty(t, runner.Calls())
	})

	t.Run("should decline the GitLab token when GITLAB_HOST names another server", func(t *testing.T) {
		t.Parallel()
		// given
		env := map[string]string{"GITLAB_TOKEN": fixtureToken, "GITLAB_HOST": "gitlab.example.com"}
		chain := newCredentialChain(env, &gitforgeDoubles.CLIRunnerStub{}, &gitforgeDoubles.AzureTenantResolverStub{})

		// when
		_, _, err := repo.ResolveProviderWith(repo.ProviderGitLab, "", chain)

		// then
		require.ErrorIs(t, err, credentialEntities.ErrNoCredential)
		require.ErrorContains(t, err, "gitlab.example.com")
	})

	t.Run("should decline GL_TOKEN when semantic-release's GL_URL names another server", func(t *testing.T) {
		t.Parallel()
		// given: the pair semantic-release's GitLab plugin documents for a self-managed server
		env := map[string]string{"GL_TOKEN": fixtureToken, "GL_URL": "https://gitlab.example.com"}
		chain := newCredentialChain(env, &gitforgeDoubles.CLIRunnerStub{}, &gitforgeDoubles.AzureTenantResolverStub{})

		// when
		_, _, err := repo.ResolveProviderWith(repo.ProviderGitLab, "", chain)

		// then
		require.ErrorIs(t, err, credentialEntities.ErrNoCredential)
		require.ErrorContains(t, err, "GL_URL names the GitLab server gitlab.example.com")
	})

	t.Run("should prefer GITHUB_TOKEN over GH_TOKEN when both are exported", func(t *testing.T) {
		t.Parallel()
		// given
		env := map[string]string{"GITHUB_TOKEN": fixtureToken, "GH_TOKEN": fixtureOtherToken}
		chain := newCredentialChain(env, &gitforgeDoubles.CLIRunnerStub{}, &gitforgeDoubles.AzureTenantResolverStub{})

		// when
		provider, _, err := repo.ResolveProviderWith(repo.ProviderGitHub, "", chain)

		// then
		require.NoError(t, err)
		assert.Equal(t, fixtureToken, provider.AuthToken())
	})

	for _, scenario := range []struct {
		providerName string
		variable     string
	}{
		{repo.ProviderGitHub, "GITHUB_TOKEN"},
		{repo.ProviderGitHub, "GH_TOKEN"},
		{repo.ProviderAzureDevOps, "AZURE_DEVOPS_EXT_PAT"},
		{repo.ProviderAzureDevOps, "SYSTEM_ACCESSTOKEN"},
		{repo.ProviderGitLab, "GITLAB_TOKEN"},
		{repo.ProviderGitLab, "GL_TOKEN"},
		{repo.ProviderCodeberg, "CODEBERG_TOKEN"},
	} {
		t.Run("should authenticate "+scenario.providerName+" with "+scenario.variable+" when only it is exported",
			func(t *testing.T) {
				t.Parallel()
				// given
				runner := &gitforgeDoubles.CLIRunnerStub{}
				env := map[string]string{scenario.variable: fixtureToken}
				chain := newCredentialChain(env, runner, &gitforgeDoubles.AzureTenantResolverStub{})

				// when
				provider, _, err := repo.ResolveProviderWith(scenario.providerName, "", chain)

				// then
				require.NoError(t, err)
				assert.Equal(t, fixtureToken, provider.AuthToken())
				assert.Empty(t, runner.Calls())
			})
	}
}

func TestResolveWithDefaultCredentialChain(t *testing.T) {
	t.Parallel()

	t.Run("should refuse an unknown provider when no resolver is given", func(t *testing.T) {
		t.Parallel()
		// given
		const providerName = "bitbucket"

		// when
		_, _, err := repo.ResolveProviderForOwner(providerName, fixtureOrganization)

		// then
		require.EqualError(t, err, "unknown provider: bitbucket")
	})

	t.Run("should refuse a provider without fork resolution when no resolver is given", func(t *testing.T) {
		t.Parallel()
		// given
		const providerName = repo.ProviderCodeberg

		// when
		_, _, _, err := repo.ResolveForkSync(providerName)

		// then
		require.ErrorContains(t, err, "fork resolution not supported")
	})
}
