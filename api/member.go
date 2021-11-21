package api

import (
	"context"
	"encoding/json"
)

type MemberStatus string

const (
	Unknown MemberStatus = "UNKNOWN"
	Invited MemberStatus = "INVITED"
	Active  MemberStatus = "ACTIVE"
)

func (e MemberStatus) String() string {
	switch e {
	case Unknown:
		return "UNKNOWN"
	case Invited:
		return "INVITED"
	case Active:
		return "ACTIVE"
	}
	return ""
}

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
	ID int `jsonapi:"primary,member"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Status      MemberStatus `jsonapi:"attr,status"`
	Role        Role         `jsonapi:"attr,role"`
	PrincipalID int
	Principal   *Principal `jsonapi:"attr,principal"`
}

type MemberCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Status      MemberStatus `jsonapi:"attr,status"`
	Role        Role         `jsonapi:"attr,role"`
	PrincipalID int          `jsonapi:"attr,principalID"`
}

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

type MemberPatch struct {
	ID int

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

type MemberService interface {
	CreateMember(ctx context.Context, create *MemberCreate) (*Member, error)
	FindMemberList(ctx context.Context, find *MemberFind) ([]*Member, error)
	FindMember(ctx context.Context, find *MemberFind) (*Member, error)
	PatchMember(ctx context.Context, patch *MemberPatch) (*Member, error)
}
