package doubles

// AzureTenantResolverStub supplies organization metadata without network access.
type AzureTenantResolverStub struct {
	Tenant string
	Err    error
}

func (s *AzureTenantResolverStub) GetTenant(_ string) (string, error) {
	return s.Tenant, s.Err
}
