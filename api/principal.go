package api

import (
	"context"
	"encoding/json"
)

const SYSTEM_BOT_ID = 1

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
	CreatorID int   `jsonapi:"attr,creatorID"`
	CreatedTs int64 `jsonapi:"attr,createdTs"`
	UpdaterID int   `jsonapi:"attr,updaterID"`
	UpdatedTs int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Type  PrincipalType `jsonapi:"attr,type"`
	Name  string        `jsonapi:"attr,name"`
	Email string        `jsonapi:"attr,email"`
	// Do not return to the client
	PasswordHash string
	// Role is stored in the member table, but we include it when returning the principal.
	// This simplifies the client code where it won't require order depenendency to fetch the related member info first.
	Role Role `jsonapi:"attr,role"`
}

// Customize the Principal Marshal method so the returned object
// can map directly to the frontend Principal object without any conversion.
func (p *Principal) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        int           `json:"id"`
		CreatorID int           `json:"creatorID"`
		CreatedTs int64         `json:"createdTs"`
		UpdaterID int           `json:"updaterID"`
		UpdatedTs int64         `json:"updatedTs"`
		Type      PrincipalType `json:"type"`
		Name      string        `json:"name"`
		Email     string        `json:"email"`
		Role      Role          `json:"role"`
	}{
		ID:        p.ID,
		CreatorID: p.CreatorID,
		CreatedTs: p.CreatedTs,
		UpdaterID: p.UpdaterID,
		UpdatedTs: p.UpdatedTs,
		Type:      p.Type,
		Name:      p.Name,
		Email:     p.Email,
		Role:      p.Role,
	})
}

type PrincipalCreate struct {
	// Standard fields
	// For signup, value is SYSTEM_BOT_ID
	// For invite, value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Type         PrincipalType
	Name         string `jsonapi:"attr,name"`
	Email        string `jsonapi:"attr,email"`
	Password     string `jsonapi:"attr,password"`
	PasswordHash string
}

type PrincipalFind struct {
	ID *int

	// Domain specific fields
	Email *string
}

func (find *PrincipalFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type PrincipalPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name         *string `jsonapi:"attr,name"`
	Password     *string `jsonapi:"attr,password"`
	PasswordHash *string
}

type PrincipalService interface {
	CreatePrincipal(ctx context.Context, create *PrincipalCreate) (*Principal, error)
	FindPrincipalList(ctx context.Context) ([]*Principal, error)
	FindPrincipal(ctx context.Context, find *PrincipalFind) (*Principal, error)
	PatchPrincipal(ctx context.Context, patch *PrincipalPatch) (*Principal, error)
}
