package api

import (
	"context"
	"encoding/json"
)

type ProjectRole string

const (
	ProjectOwner     ProjectRole = "OWNER"
	ProjectDeveloper ProjectRole = "DEVELOPER"
)

func (e ProjectRole) String() string {
	switch e {
	case ProjectOwner:
		return "OWNER"
	case ProjectDeveloper:
		return "DEVELOPER"
	}
	return ""
}

type ProjectMember struct {
	ID int `jsonapi:"primary,projectMember"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	// Just returns ProjectId otherwise would cause circular dependency.
	ProjectId int `jsonapi:"attr,projectId"`

	// Domain specific fields
	Role        string `jsonapi:"attr,role"`
	PrincipalId int
	Principal   *Principal `jsonapi:"attr,principal"`
}

type ProjectMemberCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	ProjectId int

	// Domain specific fields
	Role        ProjectRole `jsonapi:"attr,role"`
	PrincipalId int         `jsonapi:"attr,principalId"`
}

type ProjectMemberFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int

	// Related fields
	ProjectId *int
}

func (find *ProjectMemberFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type ProjectMemberPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

type ProjectMemberDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type ProjectMemberService interface {
	CreateProjectMember(ctx context.Context, create *ProjectMemberCreate) (*ProjectMember, error)
	FindProjectMemberList(ctx context.Context, find *ProjectMemberFind) ([]*ProjectMember, error)
	PatchProjectMember(ctx context.Context, patch *ProjectMemberPatch) (*ProjectMember, error)
	DeleteProjectMember(ctx context.Context, delete *ProjectMemberDelete) error
}
