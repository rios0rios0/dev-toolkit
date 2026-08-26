package repo_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
	"github.com/rios0rios0/dev-toolkit/internal/testutil/doubles"
)

func TestEnvCredentialResolver(t *testing.T) {
	t.Parallel()

	t.Run("should return the token when the provider env var is set", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &repo.EnvCredentialResolver{
			Lookup: func(key string) string {
				if key == "GH_TOKEN" {
					return "env-token"
				}
				return ""
			},
		}

		// when
		cred, err := resolver.Resolve(repo.ProviderGitHub)

		// then
		require.NoError(t, err)
		assert.Equal(t, "env-token", cred.Token)
		assert.Equal(t, "GH_TOKEN environment variable", cred.Source)
	})

	t.Run("should return an error when the provider env var is empty", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &repo.EnvCredentialResolver{Lookup: func(_ string) string { return "" }}

		// when
		_, err := resolver.Resolve(repo.ProviderGitHub)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GH_TOKEN environment variable not set")
	})

	t.Run("should return an error when the provider is unknown", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &repo.EnvCredentialResolver{Lookup: func(_ string) string { return "token" }}

		// when
		_, err := resolver.Resolve("bitbucket")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider: bitbucket")
	})

	t.Run("should fall back to the process environment when no lookup is set", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := repo.NewEnvCredentialResolver()
		resolver.Lookup = nil

		// when
		_, err := resolver.Resolve("bitbucket")

		// then
		require.Error(t, err)
	})
}

func TestCLICredentialResolver(t *testing.T) {
	t.Parallel()

	t.Run("should return the token when the gh CLI is authenticated", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCLIRunnerStub().WithToken("gh", "gho_from_cli")
		resolver := &repo.CLICredentialResolver{Runner: runner}

		// when
		cred, err := resolver.Resolve(repo.ProviderGitHub)

		// then
		require.NoError(t, err)
		assert.Equal(t, "gho_from_cli", cred.Token)
		assert.Equal(t, "gh CLI", cred.Source)
		assert.Equal(t, []string{"gh auth token"}, runner.Calls)
	})

	t.Run("should ask az for an Azure DevOps token when the provider is azuredevops", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCLIRunnerStub().WithToken("az", "aad-access-token")
		resolver := &repo.CLICredentialResolver{Runner: runner}

		// when
		cred, err := resolver.Resolve(repo.ProviderAzureDevOps)

		// then
		require.NoError(t, err)
		assert.Equal(t, "aad-access-token", cred.Token)
		assert.Equal(t, "az CLI", cred.Source)
		require.Len(t, runner.Calls, 1)
		assert.Contains(t, runner.Calls[0], "account get-access-token")
		assert.Contains(t, runner.Calls[0], "499b84ac-1321-427f-aa17-267ca6975798")
	})

	t.Run("should ask glab for a token when the provider is gitlab", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCLIRunnerStub().WithToken("glab", "glpat-from-cli")
		resolver := &repo.CLICredentialResolver{Runner: runner}

		// when
		cred, err := resolver.Resolve(repo.ProviderGitLab)

		// then
		require.NoError(t, err)
		assert.Equal(t, "glpat-from-cli", cred.Token)
		assert.Equal(t, []string{"glab auth token"}, runner.Calls)
	})

	t.Run("should return an error when the CLI is not installed", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &repo.CLICredentialResolver{Runner: doubles.NewCLIRunnerStub()}

		// when
		_, err := resolver.Resolve(repo.ProviderGitHub)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gh CLI is not installed")
	})

	t.Run("should suggest logging in when the CLI is installed but not authenticated", func(t *testing.T) {
		t.Parallel()
		// given
		runner := doubles.NewCLIRunnerStub().
			WithError("gh", errors.New("gh auth token: no oauth token found"))
		resolver := &repo.CLICredentialResolver{Runner: runner}

		// when
		_, err := resolver.Resolve(repo.ProviderGitHub)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gh auth login")
		assert.Contains(t, err.Error(), "no oauth token found")
	})

	t.Run("should return an error when the CLI returns an empty token", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &repo.CLICredentialResolver{Runner: doubles.NewCLIRunnerStub().WithInstalled("gh")}

		// when
		_, err := resolver.Resolve(repo.ProviderGitHub)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty token")
	})

	t.Run("should return an error when the provider has no CLI integration", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &repo.CLICredentialResolver{Runner: doubles.NewCLIRunnerStub()}

		// when
		_, err := resolver.Resolve(repo.ProviderCodeberg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no CLI integration for provider: codeberg")
	})

	t.Run("should fall back to the real runner when none is set", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := &repo.CLICredentialResolver{Runner: nil}

		// when
		_, err := resolver.Resolve(repo.ProviderCodeberg)

		// then
		require.Error(t, err)
	})
}

func TestProviderCLIFor(t *testing.T) {
	t.Parallel()

	t.Run("should return the gh integration for github", func(t *testing.T) {
		t.Parallel()
		// given / when
		cli, ok := repo.ProviderCLIFor(repo.ProviderGitHub)

		// then
		require.True(t, ok)
		assert.Equal(t, "gh", cli.Binary)
		assert.Equal(t, "gh auth login", cli.LoginHint)
	})

	t.Run("should report no integration for codeberg", func(t *testing.T) {
		t.Parallel()
		// given / when
		_, ok := repo.ProviderCLIFor(repo.ProviderCodeberg)

		// then
		assert.False(t, ok)
	})
}

func TestChainCredentialResolver(t *testing.T) {
	t.Parallel()

	t.Run("should prefer the environment token when both sources are available", func(t *testing.T) {
		t.Parallel()
		// given
		env := &repo.EnvCredentialResolver{Lookup: func(_ string) string { return "env-token" }}
		cli := &repo.CLICredentialResolver{Runner: doubles.NewCLIRunnerStub().WithToken("gh", "cli-token")}
		chain := repo.NewChainCredentialResolver(env, cli)

		// when
		cred, err := chain.Resolve(repo.ProviderGitHub)

		// then
		require.NoError(t, err)
		assert.Equal(t, "env-token", cred.Token)
	})

	t.Run("should fall back to the CLI when no token is exported", func(t *testing.T) {
		t.Parallel()
		// given
		env := &repo.EnvCredentialResolver{Lookup: func(_ string) string { return "" }}
		cli := &repo.CLICredentialResolver{Runner: doubles.NewCLIRunnerStub().WithToken("gh", "cli-token")}
		chain := repo.NewChainCredentialResolver(env, cli)

		// when
		cred, err := chain.Resolve(repo.ProviderGitHub)

		// then
		require.NoError(t, err)
		assert.Equal(t, "cli-token", cred.Token)
		assert.Equal(t, "gh CLI", cred.Source)
	})

	t.Run("should report every reason when no source can authenticate", func(t *testing.T) {
		t.Parallel()
		// given
		env := &repo.EnvCredentialResolver{Lookup: func(_ string) string { return "" }}
		cli := &repo.CLICredentialResolver{Runner: doubles.NewCLIRunnerStub()}
		chain := repo.NewChainCredentialResolver(env, cli)

		// when
		_, err := chain.Resolve(repo.ProviderGitHub)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no github credential available")
		assert.Contains(t, err.Error(), "GH_TOKEN environment variable not set")
		assert.Contains(t, err.Error(), "gh CLI is not installed")
	})

	t.Run("should return an error when the chain has no resolvers", func(t *testing.T) {
		t.Parallel()
		// given
		chain := repo.NewChainCredentialResolver()

		// when
		_, err := chain.Resolve(repo.ProviderGitHub)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no credential resolver configured")
	})
}

func TestDefaultCredentialResolver(t *testing.T) {
	t.Parallel()

	t.Run("should build a chain that rejects providers with no source at all", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := repo.DefaultCredentialResolver()

		// when
		_, err := resolver.Resolve("bitbucket")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no bitbucket credential available")
	})
}

func TestResolveProviderWith(t *testing.T) {
	t.Parallel()

	t.Run("should build a provider when the resolver supplies a credential", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub().
			WithCredential(repo.Credential{Token: "cli-token", Source: "gh CLI"})

		// when
		provider, err := repo.ResolveProviderWith(repo.ProviderGitHub, resolver)

		// then
		require.NoError(t, err)
		assert.Equal(t, repo.ProviderGitHub, provider.Name())
		assert.Equal(t, "cli-token", provider.AuthToken())
		assert.Equal(t, repo.ProviderGitHub, resolver.LastProvider)
	})

	t.Run("should return an error when the resolver supplies no credential", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub().WithError(errors.New("not authenticated"))

		// when
		_, err := repo.ResolveProviderWith(repo.ProviderGitHub, resolver)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not authenticated")
	})

	t.Run("should return an error when the provider is unknown", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub()

		// when
		_, err := repo.ResolveProviderWith("bitbucket", resolver)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider: bitbucket")
	})
}

func TestResolveForkResolverWith(t *testing.T) {
	t.Parallel()

	t.Run("should build a fork resolver when the credential resolves", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub()

		// when
		forkResolver, err := repo.ResolveForkResolverWith(repo.ProviderGitHub, resolver)

		// then
		require.NoError(t, err)
		assert.NotNil(t, forkResolver)
	})

	t.Run("should return an error when fork resolution is unsupported", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub()

		// when
		_, err := repo.ResolveForkResolverWith(repo.ProviderCodeberg, resolver)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fork resolution not supported")
	})

	t.Run("should return an error when the credential does not resolve", func(t *testing.T) {
		t.Parallel()
		// given
		resolver := doubles.NewCredentialResolverStub().WithError(errors.New("not authenticated"))

		// when
		_, err := repo.ResolveForkResolverWith(repo.ProviderGitHub, resolver)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not authenticated")
	})
}
