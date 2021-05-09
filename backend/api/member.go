package api

import "context"

type Role string

const (
	Owner     Role = "OWNER"
	DBA       Role = "DBA"
	Developer Role = "DEVELOPER"
)

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

type MemberPatch struct {
	// Standard fields
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

type MemberService interface {
	CreateMember(ctx context.Context, create *MemberCreate) (*Member, error)
	FindMemberList(ctx context.Context, workspaceId int) ([]*Member, error)
	PatchMemberByID(ctx context.Context, id int, patch *MemberPatch) (*Member, error)
	DeleteMemberByID(ctx context.Context, id int) error
}
