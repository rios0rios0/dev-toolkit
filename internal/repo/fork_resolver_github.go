package repo

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v66/github"
)

// GitHubForkResolver implements ForkResolver using the GitHub API.
type GitHubForkResolver struct {
	client *gh.Client
}

// NewGitHubForkResolver creates a resolver that queries GitHub for fork parent info.
func NewGitHubForkResolver(token string) *GitHubForkResolver {
	return &GitHubForkResolver{
		client: gh.NewClient(nil).WithAuthToken(token),
	}
}

func (r *GitHubForkResolver) GetParentInfo(
	ctx context.Context, owner, repoName string,
) (*ParentInfo, error) {
	repo, _, err := r.client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo %s/%s: %w", owner, repoName, err)
	}

	if repo.Parent == nil {
		return nil, fmt.Errorf("repo %s/%s has no parent (not a fork)", owner, repoName)
	}

	return &ParentInfo{
		SSHURL:        repo.Parent.GetSSHURL(),
		DefaultBranch: repo.Parent.GetDefaultBranch(),
	}, nil
}
