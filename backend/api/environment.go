package api

import "context"

type Environment struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus   RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Order int    `jsonapi:"attr,order"`
}

type EnvironmentCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

type EnvironmentFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int
}

type EnvironmentPatch struct {
	ID int `jsonapi:"primary,environmentPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	Name  *string `jsonapi:"attr,name"`
	Order *int    `jsonapi:"attr,order"`
}

type EnvironmentDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type EnvironmentService interface {
	CreateEnvironment(ctx context.Context, create *EnvironmentCreate) (*Environment, error)
	FindEnvironmentList(ctx context.Context, find *EnvironmentFind) ([]*Environment, error)
	FindEnvironment(ctx context.Context, find *EnvironmentFind) (*Environment, error)
	PatchEnvironment(ctx context.Context, patch *EnvironmentPatch) (*Environment, error)
}
