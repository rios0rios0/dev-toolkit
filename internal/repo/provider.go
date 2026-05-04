package repo

import (
	"fmt"
	"os"
	"strings"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
	adoProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/azuredevops"
	cbProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/codeberg"
	ghProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/github"
	glProvider "github.com/rios0rios0/gitforge/pkg/providers/infrastructure/gitlab"
	gitRegistry "github.com/rios0rios0/gitforge/pkg/registry/infrastructure"
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
var providerTokenEnv = map[string]string{
	ProviderGitHub:      "GH_TOKEN",
	ProviderAzureDevOps: "AZURE_DEVOPS_EXT_PAT",
	ProviderGitLab:      "GITLAB_TOKEN",
	ProviderCodeberg:    "CODEBERG_TOKEN",
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

// ProviderTokenEnv returns the environment variable name for the given provider's token.
func ProviderTokenEnv(providerName string) string {
	return providerTokenEnv[providerName]
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

// ResolveProvider creates a ForgeProvider by looking up the token from the environment.
func ResolveProvider(providerName string) (globalEntities.ForgeProvider, error) {
	envVar := providerTokenEnv[providerName]
	token := os.Getenv(envVar)
	if token == "" {
		return nil, fmt.Errorf("%s environment variable not set", envVar)
	}

	registry := NewProviderRegistry()
	provider, err := registry.Get(providerName, token)
	if err != nil {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
	return provider, nil
}
