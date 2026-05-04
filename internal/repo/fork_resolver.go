package repo

import (
	"context"
	"fmt"
	"os"
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
var resolverFactoryMap = map[string]func(token string) ForkResolver{
	ProviderGitHub: func(token string) ForkResolver { return NewGitHubForkResolver(token) },
}

// ResolveForkResolver creates a ForkResolver for the given provider using the environment token.
func ResolveForkResolver(providerName string) (ForkResolver, error) {
	factory, ok := resolverFactoryMap[providerName]
	if !ok {
		return nil, fmt.Errorf("fork resolution not supported for provider: %s", providerName)
	}

	envVar := ProviderTokenEnv(providerName)
	if envVar == "" {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	token := os.Getenv(envVar)
	if token == "" {
		return nil, fmt.Errorf("%s environment variable not set", envVar)
	}

	return factory(token), nil
}
