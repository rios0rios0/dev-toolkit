package repo

import (
	"context"
	"fmt"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
	credentialInfra "github.com/rios0rios0/gitforge/v4/pkg/credentials/infrastructure"
	globalEntities "github.com/rios0rios0/gitforge/v4/pkg/global/domain/entities"
)

// providerServiceTypeMap names each provider the way gitforge's credential chain does. It
// is the inverse of the registry's ServiceTypeToProviderName for the providers dev-toolkit
// supports: the registry and dev-toolkit use these names, while a credential request names
// its forge by ServiceType.
//
//nolint:gochecknoglobals // read-only configuration lookup table
var providerServiceTypeMap = map[string]globalEntities.ServiceType{
	ProviderGitHub:      globalEntities.GITHUB,
	ProviderAzureDevOps: globalEntities.AZUREDEVOPS,
	ProviderGitLab:      globalEntities.GITLAB,
	ProviderCodeberg:    globalEntities.CODEBERG,
}

// newCredentialResolver returns the credential chain that the Resolve functions without a
// resolver argument ask: gitforge's default chain, the provider's token environment
// variables first and then its CLI. It is a variable only so that an internal test can put a
// recording resolver in its place and check what those functions ask for -- the owner a
// request carries, which scopes an az token to the organization's tenant -- without reading
// the environment or running gh or az. Nothing else assigns it.
//
//nolint:gochecknoglobals // test seam: only credential_internal_test.go replaces it
var newCredentialResolver = func() credentialEntities.CredentialResolver {
	return credentialInfra.NewDefaultCredentialResolver()
}

// resolveCredential asks resolver for a provider's token. The owner travels as the
// request's organization: for Azure DevOps, gitforge then issues the az token in that
// organization's Microsoft Entra tenant, and every other provider ignores it. A provider
// dev-toolkit does not know is refused before any credential source is consulted.
func resolveCredential(
	providerName, owner string, resolver credentialEntities.CredentialResolver,
) (credentialEntities.Credential, error) {
	serviceType, ok := providerServiceTypeMap[providerName]
	if !ok {
		return credentialEntities.Credential{}, fmt.Errorf("unknown provider: %s", providerName)
	}
	return resolver.Resolve(context.Background(), credentialEntities.CredentialRequest{
		ServiceType:  serviceType,
		Organization: owner,
	})
}
