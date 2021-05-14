package api

import "context"

type Environment struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus   RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId   int       `jsonapi:"attr,creatorId"`
	CreatedTs   int64     `jsonapi:"attr,createdTs"`
	UpdaterId   int       `jsonapi:"attr,updaterId"`
	UpdatedTs   int64     `jsonapi:"attr,updatedTs"`
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
	PatchEnvironmentByID(ctx context.Context, patch *EnvironmentPatch) (*Environment, error)
}
