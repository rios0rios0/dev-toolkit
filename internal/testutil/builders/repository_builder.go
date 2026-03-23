package builders

import globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"

// RepositoryBuilder builds gitforge Repository entities for testing.
type RepositoryBuilder struct {
	repo globalEntities.Repository
}

func NewRepositoryBuilder() *RepositoryBuilder {
	return &RepositoryBuilder{
		repo: globalEntities.Repository{
			Name:          "repo",
			Organization:  "owner",
			DefaultBranch: "main",
			ProviderName:  "github",
		},
	}
}

func (b *RepositoryBuilder) WithName(name string) *RepositoryBuilder {
	b.repo.Name = name
	return b
}

func (b *RepositoryBuilder) WithOrganization(org string) *RepositoryBuilder {
	b.repo.Organization = org
	return b
}

func (b *RepositoryBuilder) WithProject(project string) *RepositoryBuilder {
	b.repo.Project = project
	return b
}

func (b *RepositoryBuilder) WithArchived(archived bool) *RepositoryBuilder {
	b.repo.IsArchived = archived
	return b
}

func (b *RepositoryBuilder) WithSSHURL(url string) *RepositoryBuilder {
	b.repo.SSHURL = url
	return b
}

func (b *RepositoryBuilder) WithProvider(name string) *RepositoryBuilder {
	b.repo.ProviderName = name
	return b
}

func (b *RepositoryBuilder) Build() globalEntities.Repository {
	return b.repo
}
