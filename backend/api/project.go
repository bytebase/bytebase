package api

import "context"

type Project struct {
	ID string `jsonapi:"primary,project"`

	// Related fields
	ProjectMemberList *ResourceObject `jsonapi:"relation,environment"`

	// Standard fields
	RowStatus   RowStatus `jsonapi:"attr,rowStatus"`
	WorkspaceId int
	CreatorId   int   `jsonapi:"attr,creatorId"`
	CreatedTs   int64 `jsonapi:"attr,createdTs"`
	UpdaterId   int   `jsonapi:"attr,updaterId"`
	UpdatedTs   int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Key  string `jsonapi:"attr,key"`
}

type ProjectCreate struct {
	// Standard fields
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Key  string `jsonapi:"attr,key"`
}

type ProjectFind struct {
	// Standard fields
	ID          *int
	WorkspaceId *int
}

type ProjectPatch struct {
	// Standard fields
	ID          int     `jsonapi:"primary,project-patch"`
	RowStatus   *string `jsonapi:"attr,rowStatus"`
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
	Key  *string `jsonapi:"attr,key"`
}

type ProjectService interface {
	CreateProject(ctx context.Context, create *ProjectCreate) (*Project, error)
	FindProjectList(ctx context.Context, find *ProjectFind) ([]*Project, error)
	FindProject(ctx context.Context, find *ProjectFind) (*Project, error)
	PatchProjectByID(ctx context.Context, patch *ProjectPatch) (*Project, error)
}
