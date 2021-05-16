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
	CreatorId   int
	Creator     *Principal `jsonapi:"relation,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"relation,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Domain specific fields
	Role        Role `jsonapi:"attr,role"`
	PrincipalId int
	Principal   *Principal `jsonapi:"relation,principal"`
}

type MemberCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Domain specific fields
	Role        Role `jsonapi:"attr,role"`
	PrincipalId int  `jsonapi:"attr,principalId"`
}

type MemberFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int

	// Domain specific fields
	PrincipalId *int
}

type MemberPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

type MemberDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type MemberService interface {
	CreateMember(ctx context.Context, create *MemberCreate) (*Member, error)
	FindMemberList(ctx context.Context, find *MemberFind) ([]*Member, error)
	FindMember(ctx context.Context, find *MemberFind) (*Member, error)
	PatchMember(ctx context.Context, patch *MemberPatch) (*Member, error)
	DeleteMember(ctx context.Context, delete *MemberDelete) error
}
