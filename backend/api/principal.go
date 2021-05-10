package api

import "context"

const SYSTEM_BOT_ID = 1

type PrincipalStatus string

const (
	Unknown PrincipalStatus = "UNKNOWN"
	Invited PrincipalStatus = "INVITED"
	Active  PrincipalStatus = "ACTIVE"
)

func (e PrincipalStatus) String() string {
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

type PrincipalType string

const (
	EndUser PrincipalType = "END_USER"
	BOT     PrincipalType = "BOT"
)

func (e PrincipalType) String() string {
	switch e {
	case EndUser:
		return "END_USER"
	case BOT:
		return "BOT"
	}
	return ""
}

type Principal struct {
	ID int `jsonapi:"primary,principal"`

	// Standard fields
	CreatorId int   `jsonapi:"attr,creatorId"`
	CreatedTs int64 `jsonapi:"attr,createdTs"`
	UpdaterId int   `jsonapi:"attr,updaterId"`
	UpdatedTs int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Status PrincipalStatus `jsonapi:"attr,status"`
	Type   PrincipalType   `jsonapi:"attr,type"`
	Name   string          `jsonapi:"attr,name"`
	Email  string          `jsonapi:"attr,email"`
	// Not needed to return to the client
	PasswordHash string
}

type PrincipalCreate struct {
	// Standard fields
	// For signup, value is SYSTEM_BOT_ID
	// For invite, value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Status       PrincipalStatus `jsonapi:"attr,status"`
	Type         PrincipalType
	Name         string `jsonapi:"attr,name"`
	Email        string `jsonapi:"attr,email"`
	PasswordHash string
}

type PrincipalFilter struct {
	// Standard fields
	ID *int

	// Domain specific fields
	Email *string
}

type PrincipalPatch struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
}

type PrincipalService interface {
	CreatePrincipal(ctx context.Context, create *PrincipalCreate) (*Principal, error)
	FindPrincipalList(ctx context.Context, filter *PrincipalFilter) ([]*Principal, error)
	FindPrincipal(ctx context.Context, filter *PrincipalFilter) (*Principal, error)
	PatchPrincipalByID(ctx context.Context, id int, patch *PrincipalPatch) (*Principal, error)
}
