package api

import (
	"context"
	"database/sql"
	"encoding/json"
)

const DEFAULT_PROJECT_ID = 1

type ProjectWorkflowType string

const (
	UI_WORKFLOW  ProjectWorkflowType = "UI"
	VCS_WORKFLOW ProjectWorkflowType = "VCS"
)

func (e ProjectWorkflowType) String() string {
	switch e {
	case UI_WORKFLOW:
		return "UI"
	case VCS_WORKFLOW:
		return "VCS"
	}
	return ""
}

type Project struct {
	ID int `jsonapi:"primary,project"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectMemberList []*ProjectMember `jsonapi:"relation,projectMember"`

	// Domain specific fields
	Name         string              `jsonapi:"attr,name"`
	Key          string              `jsonapi:"attr,key"`
	WorkflowType ProjectWorkflowType `jsonapi:"attr,workflowType"`
}

type ProjectCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	Key  string `jsonapi:"attr,key"`
}

type ProjectFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus

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
	UpdaterId int

	// Domain specific fields
	Name         *string              `jsonapi:"attr,name"`
	Key          *string              `jsonapi:"attr,key"`
	WorkflowType *ProjectWorkflowType `jsonapi:"attr,workflowType"`
}

type ProjectService interface {
	CreateProject(ctx context.Context, create *ProjectCreate) (*Project, error)
	FindProjectList(ctx context.Context, find *ProjectFind) ([]*Project, error)
	FindProject(ctx context.Context, find *ProjectFind) (*Project, error)
	PatchProject(ctx context.Context, patch *ProjectPatch) (*Project, error)
	// This is specifically used to update the ProjectWorkflowType when linking/unlinking the repository.
	PatchProjectWithTx(ctx context.Context, tx *sql.Tx, patch *ProjectPatch) (*Project, error)
}
