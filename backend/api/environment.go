package api

import "context"

type Environment struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus   RowStatus `jsonapi:"attr,rowStatus"`
	WorkspaceId int
	CreatorId   int   `jsonapi:"attr,creatorId"`
	CreatedTs   int64 `jsonapi:"attr,createdTs"`
	UpdaterId   int   `jsonapi:"attr,updaterId"`
	UpdatedTs   int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Order int    `jsonapi:"attr,order"`
}

type EnvironmentCreate struct {
	// Standard fields
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
}

type EnvironmentFind struct {
	// Standard fields
	WorkspaceId *int
}

type EnvironmentPatch struct {
	// Standard fields
	ID          int     `jsonapi:"primary,environment-patch"`
	RowStatus   *string `jsonapi:"attr,rowStatus"`
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name  *string `jsonapi:"attr,name"`
	Order *int    `jsonapi:"attr,order"`
}

type EnvironmentDelete struct {
	// Standard fields
	ID int
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type EnvironmentService interface {
	CreateEnvironment(ctx context.Context, create *EnvironmentCreate) (*Environment, error)
	FindEnvironmentList(ctx context.Context, find *EnvironmentFind) ([]*Environment, error)
	PatchEnvironmentByID(ctx context.Context, patch *EnvironmentPatch) (*Environment, error)
	DeleteEnvironmentByID(ctx context.Context, delete *EnvironmentDelete) error
}
