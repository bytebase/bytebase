package api

import "context"

const SYSTEM_BOT_ID = 1

type PrincipalStatus string

const (
	Unknown PrincipalStatus = "UNKNOWN"
	Invited PrincipalStatus = "INVITED"
	Active  PrincipalStatus = "ACTIVE"
)

type PrincipalType string

const (
	EndUser PrincipalType = "END_USER"
	BOT     PrincipalType = "BOT"
)

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

type PrincipalPatch struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
}

type PrincipalService interface {
	CreatePrincipal(ctx context.Context, create *PrincipalCreate) (*Principal, error)
	FindPrincipalList(ctx context.Context) ([]*Principal, error)
	FindPrincipalByEmail(ctx context.Context, email string) (*Principal, error)
	FindPrincipalByID(ctx context.Context, id int) (*Principal, error)
	PatchPrincipalByID(ctx context.Context, id int, patch *PrincipalPatch) (*Principal, error)
}
