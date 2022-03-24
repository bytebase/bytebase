package relation

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
)

var (
	_ api.MemberService = (*MemberServiceImpl)(nil)
)

// MemberServiceImpl implements the NewMemberService interface
type MemberServiceImpl struct {
	store *store.Store
}

// NewMemberServiceImpl creates a new instance of NewMemberServiceImpl
func NewMemberServiceImpl(store *store.Store) *MemberServiceImpl {
	return &MemberServiceImpl{
		store: store,
	}
}

// Create creates an instance of Member
func (s *MemberServiceImpl) Create(ctx context.Context, create *api.MemberCreate) (*api.Member, error) {
	memberRaw, err := s.store.Member.Create(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Member with MemberCreate[%+v], error[%w]", create, err)
	}
	member, err := s.store.Member.Compose(ctx, memberRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Member with MemberRaw[%+v], error[%w]", memberRaw, err)
	}
	return member, nil
}

// FindList finds a list of Member instances
func (s *MemberServiceImpl) FindList(ctx context.Context, find *api.MemberFind) ([]*api.Member, error) {
	memberRawList, err := s.store.Member.FindList(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Member list, error[%w]", err)
	}
	var memberList []*api.Member
	for _, raw := range memberRawList {
		member, err := s.store.Member.Compose(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose Member role with MemberRaw[%+v], error[%w]", raw, err)
		}
		memberList = append(memberList, member)
	}
	return memberList, nil
}

// Find finds an instance of Member
func (s *MemberServiceImpl) Find(ctx context.Context, find *api.MemberFind) (*api.Member, error) {
	memberRaw, err := s.store.Member.Find(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Member with MemberFind[%+v], error[%w]", find, err)
	}
	if memberRaw == nil {
		return nil, nil
	}
	member, err := s.store.Member.Compose(ctx, memberRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Member role with MemberRaw[%+v], error[%w]", memberRaw, err)
	}
	return member, nil
}

// Patch patches an instance of Member
func (s *MemberServiceImpl) Patch(ctx context.Context, patch *api.MemberPatch) (*api.Member, error) {
	memberRaw, err := s.store.Member.Patch(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch Member with MemberPatch[%+v], error[%w]", patch, err)
	}
	member, err := s.store.Member.Compose(ctx, memberRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Member role with MemberRaw[%+v], error[%w]", memberRaw, err)
	}
	return member, nil
}
