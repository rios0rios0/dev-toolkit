package repo

import (
	"context"
	"fmt"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
)

// ParentInfo holds the upstream parent repository's SSH URL and default branch.
type ParentInfo struct {
	SSHURL        string
	DefaultBranch string
}

// ForkResolver looks up the parent repository of a fork on a Git hosting provider.
type ForkResolver interface {
	GetParentInfo(ctx context.Context, owner, repoName string) (*ParentInfo, error)
}

//nolint:gochecknoglobals // read-only configuration lookup table
var resolverFactoryMap = map[string]func(token string) (ForkResolver, error){
	ProviderGitHub: func(token string) (ForkResolver, error) {
		resolver, err := NewGitHubForkResolver(token)
		if err != nil {
			return nil, err
		}
		return resolver, nil
	},
}

// ResolveForkSync builds what fork-sync needs from gitforge's default credential chain; see
// ResolveForkSyncWith.
func ResolveForkSync(providerName string) (globalEntities.ForgeProvider, ForkResolver, string, error) {
	return ResolveForkSyncWith(providerName, newCredentialResolver())
}

// ResolveForkSyncWith builds what fork-sync needs from the one credential the given resolver
// supplies: the provider that lists the owner's repositories, among them the forks, the
// resolver that looks up each fork's parent, and where that credential came from. A provider
// without fork resolution is refused before any credential source is consulted, so a
// directory of another provider is told so instead of running that provider's CLI first --
// for Azure DevOps, az in the default tenant, which can fail with a sign-in error that hides
// the real reason.
func ResolveForkSyncWith(
	providerName string, resolver credentialEntities.CredentialResolver,
) (globalEntities.ForgeProvider, ForkResolver, string, error) {
	newForkResolver, supported := resolverFactoryMap[providerName]
	if !supported {
		return nil, nil, "", fmt.Errorf("fork resolution not supported for provider: %s", providerName)
	}

	cred, err := resolveCredential(providerName, "", resolver)
	if err != nil {
		return nil, nil, "", err
	}

	provider, err := newProvider(providerName, cred.Token)
	if err != nil {
		return nil, nil, "", err
	}

	forkResolver, err := newForkResolver(cred.Token)
	if err != nil {
		return nil, nil, "", err
	}
	return provider, forkResolver, cred.Source, nil
}
