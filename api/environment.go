package api

import (
	"context"
	"encoding/json"
)

type Environment struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Order int    `jsonapi:"attr,order"`
}

type EnvironmentCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

type EnvironmentFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus
}

func (find *EnvironmentFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type EnvironmentPatch struct {
	ID int `jsonapi:"primary,environmentPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name  *string `jsonapi:"attr,name"`
	Order *int    `jsonapi:"attr,order"`
}

type EnvironmentDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

type EnvironmentService interface {
	CreateEnvironment(ctx context.Context, create *EnvironmentCreate) (*Environment, error)
	FindEnvironmentList(ctx context.Context, find *EnvironmentFind) ([]*Environment, error)
	FindEnvironment(ctx context.Context, find *EnvironmentFind) (*Environment, error)
	PatchEnvironment(ctx context.Context, patch *EnvironmentPatch) (*Environment, error)
}
