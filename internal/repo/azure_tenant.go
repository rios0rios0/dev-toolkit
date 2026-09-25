package repo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

// AzureTenantResolver discovers the Microsoft Entra tenant owning an organization.
type AzureTenantResolver interface {
	GetTenant(organization string) (string, error)
}

// HTTPAzureTenantResolver reads public Azure DevOps tenant metadata without sending
// credentials. BaseURL and Client may be supplied for a local HTTP test server.
type HTTPAzureTenantResolver struct {
	BaseURL string
	Client  *http.Client
}

var azureTenantIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

func (r *HTTPAzureTenantResolver) GetTenant(organization string) (string, error) {
	baseURL := r.BaseURL
	if baseURL == "" {
		baseURL = "https://dev.azure.com"
	}
	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()
	endpoint := baseURL + "/" + url.PathEscape(organization) + "/_apis/connectionData"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create tenant discovery request: %w", err)
	}
	client := http.Client{}
	if r.Client != nil {
		client = *r.Client
	}
	// Authentication challenges and sign-in redirects carry the metadata themselves.
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request organization tenant: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("tenant discovery returned HTTP %d", resp.StatusCode)
	}
	tenant := resp.Header.Get("X-Vss-Resourcetenant")
	// Personal Microsoft account organizations may have no Entra tenant.
	if tenant == "" || tenant == "00000000-0000-0000-0000-000000000000" {
		return "", nil
	}
	if !azureTenantIDPattern.MatchString(tenant) {
		return "", errors.New("tenant discovery returned an invalid tenant ID")
	}
	return tenant, nil
}
