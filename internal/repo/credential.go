package repo

import (
	"context"
	"fmt"

	credentialEntities "github.com/rios0rios0/gitforge/v4/pkg/credentials/domain/entities"
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
