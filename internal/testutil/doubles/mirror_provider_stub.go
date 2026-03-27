package doubles

import (
	"context"

	globalEntities "github.com/rios0rios0/gitforge/pkg/global/domain/entities"
)

// MirrorProviderStub is a test double that implements both ForgeProvider and MirrorProvider.
type MirrorProviderStub struct {
	ForgeProviderStub
	MigrateFunc func(ctx context.Context, input globalEntities.MirrorInput) error
}

func NewMirrorProviderStub() *MirrorProviderStub {
	return &MirrorProviderStub{
		ForgeProviderStub: *NewForgeProviderStub(),
		MigrateFunc: func(_ context.Context, _ globalEntities.MirrorInput) error {
			return nil
		},
	}
}

func (s *MirrorProviderStub) WithMigrateError(err error) *MirrorProviderStub {
	s.MigrateFunc = func(_ context.Context, _ globalEntities.MirrorInput) error {
		return err
	}
	return s
}

func (s *MirrorProviderStub) MigrateRepository(
	ctx context.Context, input globalEntities.MirrorInput,
) error {
	return s.MigrateFunc(ctx, input)
}
