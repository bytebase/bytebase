package relation

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
)

var (
	_ api.PrincipalService = (*PrincipalServiceImpl)(nil)
)

// PrincipalServiceImpl implements the NewPrincipalService interface
type PrincipalServiceImpl struct {
	store *store.Store
}

// NewPrincipalServiceImpl creates a new instance of NewPrincipalServiceImpl
func NewPrincipalServiceImpl(store *store.Store) *PrincipalServiceImpl {
	return &PrincipalServiceImpl{
		store: store,
	}
}

// Create creates an instance of Principal
func (s *PrincipalServiceImpl) Create(ctx context.Context, create *api.PrincipalCreate) (*api.Principal, error) {
	principalRaw, err := s.store.Principal.Create(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Principal with PrincipalCreate[%+v], error[%w]", create, err)
	}
	// TODO(dragonly): compose principal role
	principal := principalRaw.ToPrincipal()
	return principal, nil
}

// FindList finds a list of Principal instances
func (s *PrincipalServiceImpl) FindList(ctx context.Context) ([]*api.Principal, error) {
	principalRawList, err := s.store.Principal.FindList(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Principal list, error[%w]", err)
	}
	var principalList []*api.Principal
	for _, raw := range principalRawList {
		principal, err := s.store.Principal.Compose(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose Principal role with PrincipalRaw[%+v], error[%w]", raw, err)
		}
		principalList = append(principalList, principal)
	}
	return principalList, nil
}

// Find finds an instance of Principal
func (s *PrincipalServiceImpl) Find(ctx context.Context, find *api.PrincipalFind) (*api.Principal, error) {
	principalRaw, err := s.store.Principal.Find(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Principal with PrincipalFind[%+v], error[%w]", find, err)
	}
	if principalRaw == nil {
		return nil, nil
	}
	principal, err := s.store.Principal.Compose(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with PrincipalRaw[%+v], error[%w]", principalRaw, err)
	}
	return principal, nil
}

// Patch patches an instance of Principal
func (s *PrincipalServiceImpl) Patch(ctx context.Context, patch *api.PrincipalPatch) (*api.Principal, error) {
	principalRaw, err := s.store.Principal.Patch(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch Principal with PrincipalPatch[%+v], error[%w]", patch, err)
	}
	principal, err := s.store.Principal.Compose(ctx, principalRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Principal role with PrincipalRaw[%+v], error[%w]", principalRaw, err)
	}
	return principal, nil
}
