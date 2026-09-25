package repo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rios0rios0/dev-toolkit/internal/repo"
)

func TestHTTPAzureTenantResolver(t *testing.T) {
	t.Parallel()

	const tenantID = "11111111-2222-3333-4444-555555555555"
	for _, scenario := range []struct {
		name   string
		status int
		header string
		want   string
		err    string
	}{
		{"should discover the tenant when metadata succeeds", http.StatusOK, tenantID, tenantID, ""},
		{"should read the tenant without following a sign-in redirect", http.StatusFound, tenantID, tenantID, ""},
		{"should read the tenant from an authentication challenge", http.StatusUnauthorized, tenantID, tenantID, ""},
		{"should use the default account when there is no tenant header", http.StatusOK, "", "", ""},
		{"should use the default account for a personal organization", http.StatusOK,
			"00000000-0000-0000-0000-000000000000", "", ""},
		{"should reject invalid tenant metadata", http.StatusOK, "invalid-tenant", "", "invalid tenant ID"},
		{"should reject an unavailable organization", http.StatusNotFound, "", "", "HTTP 404"},
		{"should reject server errors even with a tenant header", http.StatusServiceUnavailable, tenantID, "", "HTTP 503"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			// given
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, "/test organization/_apis/connectionData", req.URL.Path)
				assert.Empty(t, req.Header.Get("Authorization"))
				w.Header().Set("X-Vss-Resourcetenant", scenario.header)
				w.Header().Set("Location", "/unexpected-login")
				w.WriteHeader(scenario.status)
			}))
			t.Cleanup(server.Close)
			resolver := &repo.HTTPAzureTenantResolver{BaseURL: server.URL, Client: server.Client()}

			// when
			tenant, err := resolver.GetTenant("test organization")

			// then
			if scenario.err != "" {
				require.ErrorContains(t, err, scenario.err)
				assert.Empty(t, tenant)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, scenario.want, tenant)
		})
	}

	t.Run("should report network failures when metadata cannot be reached", func(t *testing.T) {
		t.Parallel()
		// given
		server := httptest.NewServer(http.NotFoundHandler())
		server.Close()
		resolver := &repo.HTTPAzureTenantResolver{BaseURL: server.URL}

		// when
		tenant, err := resolver.GetTenant("organization")

		// then
		require.ErrorContains(t, err, "request organization tenant")
		assert.Empty(t, tenant)
	})
}
