package gist

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v93/github"
	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	credentialInfra "github.com/rios0rios0/gitforge/v4/pkg/credentials/infrastructure"
	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
)

// Provider lists gists for a given owner.
type Provider interface {
	ListGists(ctx context.Context, owner string) ([]Gist, error)
}

// GitHubProvider fetches gists from the GitHub REST API.
type GitHubProvider struct {
	client *gh.Client
}

// NewGitHubProvider builds a GitHub gist provider using a personal access token.
func NewGitHubProvider(token string) (*GitHubProvider, error) {
	client, err := gh.NewClient(gh.WithAuthToken(token))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	return &GitHubProvider{client: client}, nil
}

// ListGists paginates through all gists belonging to owner.
func (p *GitHubProvider) ListGists(ctx context.Context, owner string) ([]Gist, error) {
	const pageSize = 100
	opts := &gh.GistListOptions{PerPage: pageSize}

	var gists []Gist
	for {
		page, resp, err := p.client.Gists.List(ctx, owner, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list gists for %s: %w", owner, err)
		}
		for _, g := range page {
			gists = append(gists, Gist{
				ID:          g.GetID(),
				Owner:       owner,
				Description: g.GetDescription(),
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return gists, nil
}

// ResolveProvider builds a Provider using gitforge's default GitHub credential chain:
// GH_TOKEN or GITHUB_TOKEN when one is exported, and otherwise the github.com token held
// by an authenticated "gh" CLI. Like ResolveProviderWith, it also returns where the
// credential came from.
func ResolveProvider() (Provider, string, error) {
	return ResolveProviderWith(credentialInfra.NewDefaultCredentialResolver())
}

// ResolveProviderWith builds a Provider from whatever credential the given resolver
// supplies for GitHub. It also returns the credential's source ("GH_TOKEN environment
// variable", "gh CLI"), so the command can say which credential it works with; the token
// itself stays inside the provider.
func ResolveProviderWith(resolver credentialEntities.CredentialResolver) (Provider, string, error) {
	cred, err := resolver.Resolve(context.Background(), credentialEntities.CredentialRequest{
		ServiceType: globalEntities.GITHUB,
	})
	if err != nil {
		return nil, "", err
	}
	provider, err := NewGitHubProvider(cred.Token)
	if err != nil {
		return nil, "", err
	}
	return provider, cred.Source, nil
}
