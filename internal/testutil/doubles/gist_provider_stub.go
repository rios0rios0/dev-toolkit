package doubles

import (
	"context"

	"github.com/rios0rios0/dev-toolkit/internal/gist"
)

// GistProviderStub is a configurable test double for gist.Provider.
type GistProviderStub struct {
	Gists     []gist.Gist
	ListErr   error
	LastOwner string
}

// NewGistProviderStub creates a stub with no gists and no error.
func NewGistProviderStub() *GistProviderStub {
	return &GistProviderStub{}
}

func (s *GistProviderStub) WithGists(gs []gist.Gist) *GistProviderStub {
	s.Gists = gs
	return s
}

func (s *GistProviderStub) WithListError(err error) *GistProviderStub {
	s.ListErr = err
	return s
}

func (s *GistProviderStub) ListGists(_ context.Context, owner string) ([]gist.Gist, error) {
	s.LastOwner = owner
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	return s.Gists, nil
}
