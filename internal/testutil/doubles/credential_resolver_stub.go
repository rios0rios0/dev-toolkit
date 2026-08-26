package doubles

import (
	"github.com/rios0rios0/dev-toolkit/internal/repo"
)

// CredentialResolverStub is a configurable test double for repo.CredentialResolver.
type CredentialResolverStub struct {
	Credential   repo.Credential
	Err          error
	LastProvider string
}

// NewCredentialResolverStub creates a stub that resolves a placeholder token.
func NewCredentialResolverStub() *CredentialResolverStub {
	return &CredentialResolverStub{
		Credential: repo.Credential{Token: "stub-token", Source: "stub"},
	}
}

// WithCredential sets the credential the stub resolves.
func (s *CredentialResolverStub) WithCredential(cred repo.Credential) *CredentialResolverStub {
	s.Credential = cred
	s.Err = nil
	return s
}

// WithError makes the stub decline to resolve a credential.
func (s *CredentialResolverStub) WithError(err error) *CredentialResolverStub {
	s.Err = err
	return s
}

func (s *CredentialResolverStub) Resolve(providerName string) (repo.Credential, error) {
	s.LastProvider = providerName
	if s.Err != nil {
		return repo.Credential{}, s.Err
	}
	return s.Credential, nil
}
