package gist

import (
	"context"
	"errors"
	"fmt"
	"os"

	gh "github.com/google/go-github/v66/github"
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
	opts := &gh.GistListOptions{ListOptions: gh.ListOptions{PerPage: pageSize}}

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

// ResolveProvider builds a Provider, reading the GH_TOKEN environment variable.
func ResolveProvider() (Provider, error) {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		return nil, errors.New("GH_TOKEN environment variable not set")
	}
	return NewGitHubProvider(token), nil
}
