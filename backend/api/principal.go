package api

import (
	"context"
	"encoding/json"
	"strconv"
)

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
	// No need to return to the client
	PasswordHash string
	// From member
	Role Role
}

// Customize the Principal Marshal method so the returned object
// can map directly to the frontend Principal object without any conversion.
func (p *Principal) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        string          `json:"id"`
		CreatorId string          `json:"creatorId"`
		CreatedTs int64           `json:"createdTs"`
		UpdaterId string          `json:"updaterId"`
		UpdatedTs int64           `json:"updatedTs"`
		Status    PrincipalStatus `json:"status"`
		Type      PrincipalType   `json:"type"`
		Name      string          `json:"name"`
		Email     string          `json:"email"`
		Role      Role            `json:"role"`
	}{
		ID:        strconv.Itoa(p.ID),
		CreatorId: strconv.Itoa(p.CreatorId),
		CreatedTs: p.CreatedTs,
		UpdaterId: strconv.Itoa(p.UpdaterId),
		UpdatedTs: p.UpdatedTs,
		Status:    p.Status,
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
	CreatorId int

	// Domain specific fields
	Status       PrincipalStatus `jsonapi:"attr,status"`
	Type         PrincipalType
	Name         string `jsonapi:"attr,name"`
	Email        string `jsonapi:"attr,email"`
	PasswordHash string
}

type PrincipalFind struct {
	ID *int

	// Domain specific fields
	Email *string
}

type PrincipalPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
}

type PrincipalService interface {
	CreatePrincipal(ctx context.Context, create *PrincipalCreate) (*Principal, error)
	FindPrincipalList(ctx context.Context, find *PrincipalFind) ([]*Principal, error)
	FindPrincipal(ctx context.Context, find *PrincipalFind) (*Principal, error)
	PatchPrincipal(ctx context.Context, patch *PrincipalPatch) (*Principal, error)
}
