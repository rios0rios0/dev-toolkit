package repo

import (
	"context"
	"fmt"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	credentialInfra "github.com/rios0rios0/gitforge/v4/pkg/credentials/infrastructure"
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

// ResolveForkResolver creates a ForkResolver for the given provider using gitforge's
// default credential chain.
func ResolveForkResolver(providerName string) (ForkResolver, error) {
	return ResolveForkResolverWith(providerName, credentialInfra.NewDefaultCredentialResolver())
}

// ResolveForkResolverWith creates a ForkResolver from whatever credential the given
// resolver supplies.
func ResolveForkResolverWith(
	providerName string, resolver credentialEntities.CredentialResolver,
) (ForkResolver, error) {
	factory, ok := resolverFactoryMap[providerName]
	if !ok {
		return nil, fmt.Errorf("fork resolution not supported for provider: %s", providerName)
	}

	cred, err := resolveCredential(providerName, "", resolver)
	if err != nil {
		return nil, err
	}

	return factory(cred.Token)
}
