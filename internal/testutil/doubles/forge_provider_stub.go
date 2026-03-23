package doubles

import (
	"context"
	"fmt"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
)

// ForgeProviderStub is a test double for globalEntities.ForgeProvider.
type ForgeProviderStub struct {
	DiscoverFunc func(ctx context.Context, org string) ([]globalEntities.Repository, error)
	SSHCloneFunc func(repo globalEntities.Repository, sshAlias string) string
	NameValue    string
	AuthTokenVal string
}

func NewForgeProviderStub() *ForgeProviderStub {
	return &ForgeProviderStub{
		DiscoverFunc: func(_ context.Context, _ string) ([]globalEntities.Repository, error) {
			return nil, nil
		},
		SSHCloneFunc: func(r globalEntities.Repository, alias string) string {
			return fmt.Sprintf("git@host-%s:%s/%s.git", alias, r.Organization, r.Name)
		},
		NameValue: "github",
	}
}

func (s *ForgeProviderStub) WithRepos(repos []globalEntities.Repository) *ForgeProviderStub {
	s.DiscoverFunc = func(_ context.Context, _ string) ([]globalEntities.Repository, error) {
		return repos, nil
	}
	return s
}

func (s *ForgeProviderStub) WithDiscoverError(err error) *ForgeProviderStub {
	s.DiscoverFunc = func(_ context.Context, _ string) ([]globalEntities.Repository, error) {
		return nil, err
	}
	return s
}

func (s *ForgeProviderStub) Name() string                                { return s.NameValue }
func (s *ForgeProviderStub) MatchesURL(_ string) bool                    { return false }
func (s *ForgeProviderStub) AuthToken() string                           { return s.AuthTokenVal }
func (s *ForgeProviderStub) CloneURL(_ globalEntities.Repository) string { return "" }

func (s *ForgeProviderStub) SSHCloneURL(repo globalEntities.Repository, sshAlias string) string {
	return s.SSHCloneFunc(repo, sshAlias)
}

func (s *ForgeProviderStub) DiscoverRepositories(
	ctx context.Context, org string,
) ([]globalEntities.Repository, error) {
	return s.DiscoverFunc(ctx, org)
}

func (s *ForgeProviderStub) CreatePullRequest(
	_ context.Context, _ globalEntities.Repository, _ globalEntities.PullRequestInput,
) (*globalEntities.PullRequest, error) {
	return &globalEntities.PullRequest{}, nil
}

func (s *ForgeProviderStub) PullRequestExists(
	_ context.Context, _ globalEntities.Repository, _ string,
) (bool, error) {
	return false, nil
}
