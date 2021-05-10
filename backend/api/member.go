package api

import "context"

type Role string

const (
	Owner     Role = "OWNER"
	DBA       Role = "DBA"
	Developer Role = "DEVELOPER"
)

func (e Role) String() string {
	switch e {
	case Owner:
		return "OWNER"
	case DBA:
		return "DBA"
	case Developer:
		return "DEVELOPER"
	}
	return ""
}

type Member struct {
	ID int `jsonapi:"primary,principal"`

	// Standard fields
	WorkspaceId int
	CreatorId   int   `jsonapi:"attr,creatorId"`
	CreatedTs   int64 `jsonapi:"attr,createdTs"`
	UpdaterId   int   `jsonapi:"attr,updaterId"`
	UpdatedTs   int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Role        Role `jsonapi:"attr,role"`
	PrincipalId int  `jsonapi:"attr,principalId"`
}

type MemberCreate struct {
	// Standard fields
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Role        Role `jsonapi:"attr,role"`
	PrincipalId int  `jsonapi:"attr,principalId"`
}

type MemberFilter struct {
	// Standard fields
	ID          *int
	WorkspaceId *int

	// Domain specific fields
	PrincipalId *int
}

type MemberPatch struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

type MemberDelete struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type MemberService interface {
	CreateMember(ctx context.Context, create *MemberCreate) (*Member, error)
	FindMemberList(ctx context.Context, filter *MemberFilter) ([]*Member, error)
	FindMember(ctx context.Context, filter *MemberFilter) (*Member, error)
	PatchMemberByID(ctx context.Context, id int, patch *MemberPatch) (*Member, error)
	DeleteMemberByID(ctx context.Context, id int, delete *MemberDelete) error
}
