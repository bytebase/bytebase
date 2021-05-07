package api

import "context"

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
	ID        uint            `jsonapi:"primary,principal"`
	CreatorId uint            `jsonapi:"attr,creatorId"`
	CreatorTs int64           `jsonapi:"attr,creatorTs"`
	UpdaterId uint            `jsonapi:"attr,updaterId"`
	UpdatedTs int64           `jsonapi:"attr,updatedTs"`
	Status    PrincipalStatus `jsonapi:"attr,status"`
	Type      PrincipalType   `jsonapi:"attr,type"`
	Name      string          `jsonapi:"attr,name"`
	Email     string          `jsonapi:"attr,email"`
	// Not needed to return to the client
	PasswordHash string
}

type PrincipalService interface {
	FindPrincipalList(ctx context.Context) ([]*Principal, error)
	FindPrincipalByEmail(ctx context.Context, email string) (*Principal, error)
}
