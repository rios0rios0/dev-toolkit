package gist

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v66/github"
	"github.com/rios0rios0/dev-toolkit/internal/repo"
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
func NewGitHubProvider(token string) *GitHubProvider {
	return &GitHubProvider{client: gh.NewClient(nil).WithAuthToken(token)}
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

// ResolveProvider builds a Provider using the default GitHub credential chain: the
// GH_TOKEN environment variable when it is exported, and otherwise the token held by an
// authenticated "gh" CLI.
func ResolveProvider() (Provider, error) {
	return ResolveProviderWith(repo.DefaultCredentialResolver())
}

// ResolveProviderWith builds a Provider from whatever credential the given resolver
// supplies for GitHub.
func ResolveProviderWith(resolver repo.CredentialResolver) (Provider, error) {
	cred, err := resolver.Resolve(repo.ProviderGitHub)
	if err != nil {
		return nil, err
	}
	return NewGitHubProvider(cred.Token), nil
}
