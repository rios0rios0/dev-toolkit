package repo

import (
	"fmt"
	"strings"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
	adoProvider "github.com/rios0rios0/gitforge/v4/pkg/providers/infrastructure/azuredevops"
	cbProvider "github.com/rios0rios0/gitforge/v4/pkg/providers/infrastructure/codeberg"
	ghProvider "github.com/rios0rios0/gitforge/v4/pkg/providers/infrastructure/github"
	glProvider "github.com/rios0rios0/gitforge/v4/pkg/providers/infrastructure/gitlab"
	gitRegistry "github.com/rios0rios0/gitforge/v4/pkg/registry/infrastructure"
)

const (
	ScanDepthNested = 2
	splitOwnerLimit = 2
	DirPermissions  = 0o750
)

//nolint:gochecknoglobals // read-only configuration lookup table
var providerPathMap = map[string]string{
	"github.com":    ProviderGitHub,
	"dev.azure.com": ProviderAzureDevOps,
	"gitlab.com":    ProviderGitLab,
	"codeberg.org":  ProviderCodeberg,
}

//nolint:gochecknoglobals // read-only configuration lookup table
var providerScanDepth = map[string]int{
	ProviderGitHub:      1,
	ProviderAzureDevOps: ScanDepthNested,
	ProviderGitLab:      1,
	ProviderCodeberg:    1,
}

//nolint:gochecknoglobals // read-only configuration lookup table
var providerHostMap = map[string]string{
	ProviderGitHub:      "github.com",
	ProviderAzureDevOps: "dev.azure.com",
	ProviderGitLab:      "gitlab.com",
	ProviderCodeberg:    "codeberg.org",
}

// DetectProviderAndOwner parses a root directory path to determine the Git provider and owner.
func DetectProviderAndOwner(rootDir string) (string, string, error) {
	for pathSegment, name := range providerPathMap {
		idx := strings.Index(rootDir, "/"+pathSegment+"/")
		if idx < 0 {
			continue
		}
		after := rootDir[idx+len("/"+pathSegment+"/"):]
		parts := strings.SplitN(after, "/", splitOwnerLimit)
		if parts[0] == "" {
			return "", "", fmt.Errorf("could not extract owner from path: %s", rootDir)
		}
		return name, parts[0], nil
	}
	return "", "", fmt.Errorf("could not detect provider from path: %s", rootDir)
}

// ProviderScanDepth returns the directory scan depth for the given provider.
func ProviderScanDepth(providerName string) int {
	return providerScanDepth[providerName]
}

// ProviderHost returns the alias hostname for the given provider (used in SSH config aliases).
func ProviderHost(providerName string) string {
	return providerHostMap[providerName]
}

// Key returns the local directory key for a repository.
func Key(r globalEntities.Repository) string {
	if r.Project != "" {
		return r.Project + "/" + r.Name
	}
	return r.Name
}

// NewProviderRegistry creates a registry with all supported provider factories.
func NewProviderRegistry() *gitRegistry.ProviderRegistry {
	r := gitRegistry.NewProviderRegistry()
	r.RegisterFactory(ProviderGitHub, ghProvider.NewProvider)
	r.RegisterFactory(ProviderAzureDevOps, adoProvider.NewProvider)
	r.RegisterFactory(ProviderGitLab, glProvider.NewProvider)
	r.RegisterFactory(ProviderCodeberg, cbProvider.NewProvider)
	return r
}

// ResolveProviderForOwner creates a ForgeProvider for the repositories of one owner using
// gitforge's default credential chain: the provider's token environment variables first, then
// its own CLI, so an authenticated provider CLI is enough when no token is exported. The owner
// travels as the credential request's organization: for Azure DevOps the az token is then
// issued in that organization's tenant. An exported token still takes precedence and skips
// the tenant discovery entirely.
func ResolveProviderForOwner(providerName, owner string) (globalEntities.ForgeProvider, error) {
	return ResolveProviderWith(providerName, owner, newCredentialResolver())
}

// ResolveProviderWith creates a ForgeProvider from whatever credential the given resolver
// supplies, asking for it on behalf of owner ("" for none).
func ResolveProviderWith(
	providerName, owner string, resolver credentialEntities.CredentialResolver,
) (globalEntities.ForgeProvider, error) {
	cred, credErr := resolveCredential(providerName, owner, resolver)
	if credErr != nil {
		return nil, credErr
	}
	return newProvider(providerName, cred.Token)
}

// newProvider builds the named provider's API client around token.
func newProvider(providerName, token string) (globalEntities.ForgeProvider, error) {
	provider, err := NewProviderRegistry().Get(providerName, token)
	if err != nil {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
	return provider, nil
}
