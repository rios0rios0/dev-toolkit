package doubles

import (
	"context"

	"github.com/rios0rios0/devforge/internal/repo"
)

// ForkResolverStub is a test double for repo.ForkResolver.
type ForkResolverStub struct {
	GetParentInfoFunc func(ctx context.Context, owner, repoName string) (*repo.ParentInfo, error)
}

func NewForkResolverStub() *ForkResolverStub {
	return &ForkResolverStub{
		GetParentInfoFunc: func(_ context.Context, _, _ string) (*repo.ParentInfo, error) {
			return &repo.ParentInfo{
				SSHURL:        "git@github.com:upstream-org/repo.git",
				DefaultBranch: "main",
			}, nil
		},
	}
}

func (s *ForkResolverStub) WithParentInfo(info *repo.ParentInfo) *ForkResolverStub {
	s.GetParentInfoFunc = func(_ context.Context, _, _ string) (*repo.ParentInfo, error) {
		return info, nil
	}
	return s
}

func (s *ForkResolverStub) WithError(err error) *ForkResolverStub {
	s.GetParentInfoFunc = func(_ context.Context, _, _ string) (*repo.ParentInfo, error) {
		return nil, err
	}
	return s
}

func (s *ForkResolverStub) GetParentInfo(
	ctx context.Context, owner, repoName string,
) (*repo.ParentInfo, error) {
	return s.GetParentInfoFunc(ctx, owner, repoName)
}
