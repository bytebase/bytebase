package api

import (
	"context"
	"encoding/json"
)

const DEFAULT_PROJECT_ID = 1

type Project struct {
	ID int `jsonapi:"primary,project"`

	// Standard fields
	RowStatus   RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	ProjectMemberList []*ProjectMember `jsonapi:"relation,projectMember"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Key  string `jsonapi:"attr,key"`
}

type ProjectCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Key  string `jsonapi:"attr,key"`
}

type ProjectFind struct {
	ID *int

	// Standard fields
	RowStatus   *RowStatus
	WorkspaceId *int

	// Domain specific fields
	// If present, will only find project containing PrincipalId as a member
	PrincipalId *int
}

func (find *ProjectFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type ProjectPatch struct {
	ID int `jsonapi:"primary,projectPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
	Key  *string `jsonapi:"attr,key"`
}

type ProjectService interface {
	CreateProject(ctx context.Context, create *ProjectCreate) (*Project, error)
	FindProjectList(ctx context.Context, find *ProjectFind) ([]*Project, error)
	FindProject(ctx context.Context, find *ProjectFind) (*Project, error)
	PatchProject(ctx context.Context, patch *ProjectPatch) (*Project, error)
}
