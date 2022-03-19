package api

import (
	"context"
	"encoding/json"
)

// MemberStatus is the status of an member.
type MemberStatus string

const (
	// Unknown is the member status for UNKNOWN.
	Unknown MemberStatus = "UNKNOWN"
	// Invited is the member status for INVITED.
	Invited MemberStatus = "INVITED"
	// Active is the member status for ACTIVE.
	Active MemberStatus = "ACTIVE"
)

// Role is the type of a role.
type Role string

const (
	// Owner is the OWNER role.
	Owner Role = "OWNER"
	// DBA is the DBA role.
	DBA Role = "DBA"
	// Developer is the DEVELOPER role.
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

// MemberRaw is the store model for an Member.
// Fields have exactly the same meanings as Member.
type MemberRaw struct {
	ID int

	// Standard fields
	RowStatus RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Status      MemberStatus
	Role        Role
	PrincipalID int
}

// ToMember creates an instance of Member based on the MemberRaw.
// This is intended to be called when we need to compose an Member relationship.
func (raw *MemberRaw) ToMember() *Member {
	return &Member{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Domain specific fields
		Status:      raw.Status,
		Role:        raw.Role,
		PrincipalID: raw.PrincipalID,
	}
}

// Member is the API message for a member.
type Member struct {
	ID int `jsonapi:"primary,member"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Status      MemberStatus `jsonapi:"attr,status"`
	Role        Role         `jsonapi:"attr,role"`
	PrincipalID int
	Principal   *Principal `jsonapi:"relation,principal"`
}

// MemberCreate is the API message for creating a member.
type MemberCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Status      MemberStatus `jsonapi:"attr,status"`
	Role        Role         `jsonapi:"attr,role"`
	PrincipalID int          `jsonapi:"attr,principalId"`
}

// MemberFind is the API message for finding members.
type MemberFind struct {
	ID *int

	// Domain specific fields
	PrincipalID *int
	Role        *Role
}

func (find *MemberFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// MemberPatch is the API message for patching a member.
type MemberPatch struct {
	ID int

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

// MemberService is the service for members.
type MemberService interface {
	CreateMember(ctx context.Context, create *MemberCreate) (*MemberRaw, error)
	FindMemberList(ctx context.Context, find *MemberFind) ([]*MemberRaw, error)
	FindMember(ctx context.Context, find *MemberFind) (*MemberRaw, error)
	PatchMember(ctx context.Context, patch *MemberPatch) (*MemberRaw, error)
}
