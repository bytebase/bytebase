package api

import (
	"context"
	"database/sql"
	"encoding/json"
)

// DefaultProjectID is the ID for the default project.
const DefaultProjectID = 1

// ProjectWorkflowType is the workflow type for projects.
type ProjectWorkflowType string

const (
	// UIWorkflow is the UI workflow.
	UIWorkflow ProjectWorkflowType = "UI"
	// VCSWorkflow is the VCS workflow.
	VCSWorkflow ProjectWorkflowType = "VCS"
)

func (e ProjectWorkflowType) String() string {
	switch e {
	case UIWorkflow:
		return "UI"
	case VCSWorkflow:
		return "VCS"
	}
	return ""
}

// ProjectVisibility is the visibility of a project.
type ProjectVisibility string

const (
	// Public is the project visibility for PUBLIC.
	Public ProjectVisibility = "PUBLIC"
	// Private is the project visibility for PRIVATE.
	Private ProjectVisibility = "PRIVATE"
)

func (e ProjectVisibility) String() string {
	switch e {
	case Public:
		return "PUBLIC"
	case Private:
		return "PRIVATE"
	}
	return ""
}

// ProjectTenantMode is the tenant mode setting for project.
type ProjectTenantMode string

const (
	// TenantModeDisabled is the DISABLED value for ProjectTenantMode.
	TenantModeDisabled ProjectTenantMode = "DISABLED"
	// TenantModeTenant is the TENANT value for ProjectTenantMode.
	TenantModeTenant ProjectTenantMode = "TENANT"
)

// Project is the API message for a project.
type Project struct {
	ID int `jsonapi:"primary,project"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectMemberList []*ProjectMember `jsonapi:"relation,projectMember"`

	// Domain specific fields
	Name         string              `jsonapi:"attr,name"`
	Key          string              `jsonapi:"attr,key"`
	WorkflowType ProjectWorkflowType `jsonapi:"attr,workflowType"`
	Visibility   ProjectVisibility   `jsonapi:"attr,visibility"`
	TenantMode   ProjectTenantMode   `jsonapi:"attr,tenantMode"`
	// DBNameTemplate is only used when a project is in tenant mode.
	// Empty value means {{DB_NAME}}.
	DBNameTemplate string `jsonapi:"attr,dbNameTemplate"`
}

// ProjectCreate is the API message for creating a project.
type ProjectCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name           string            `jsonapi:"attr,name"`
	Key            string            `jsonapi:"attr,key"`
	TenantMode     ProjectTenantMode `jsonapi:"attr,tenantMode"`
	DBNameTemplate string            `jsonapi:"attr,dbNameTemplate"`
}

// ProjectFind is the API message for finding projects.
type ProjectFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus

	// Domain specific fields
	// If present, will only find project containing PrincipalID as a member
	PrincipalID *int
}

func (find *ProjectFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// ProjectPatch is the API message for patching a project.
type ProjectPatch struct {
	ID int `jsonapi:"primary,projectPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name           *string              `jsonapi:"attr,name"`
	Key            *string              `jsonapi:"attr,key"`
	WorkflowType   *ProjectWorkflowType `jsonapi:"attr,workflowType"`
	TenantMode     *ProjectTenantMode   `jsonapi:"attr,tenantMode"`
	DBNameTemplate *string              `jsonapi:"attr,dbNameTemplate"`
}

// ProjectService is the service for projects.
type ProjectService interface {
	CreateProject(ctx context.Context, create *ProjectCreate) (*Project, error)
	FindProjectList(ctx context.Context, find *ProjectFind) ([]*Project, error)
	FindProject(ctx context.Context, find *ProjectFind) (*Project, error)
	PatchProject(ctx context.Context, patch *ProjectPatch) (*Project, error)
	// This is specifically used to update the ProjectWorkflowType when linking/unlinking the repository.
	PatchProjectTx(ctx context.Context, tx *sql.Tx, patch *ProjectPatch) (*Project, error)
}
