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
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns ProjectID otherwise would cause circular dependency.
	ProjectID int `jsonapi:"attr,projectId"`

	// Domain specific fields
	Role        string `jsonapi:"attr,role"`
	PrincipalID int
	Principal   *Principal `jsonapi:"attr,principal"`
}

type ProjectMemberCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID int

	// Domain specific fields
	Role        ProjectRole `jsonapi:"attr,role"`
	PrincipalID int         `jsonapi:"attr,principalId"`
}

type ProjectMemberFind struct {
	ID *int

	// Related fields
	ProjectID *int
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
	UpdaterID int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

type ProjectMemberDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

type ProjectMemberService interface {
	CreateProjectMember(ctx context.Context, create *ProjectMemberCreate) (*ProjectMember, error)
	FindProjectMemberList(ctx context.Context, find *ProjectMemberFind) ([]*ProjectMember, error)
	FindProjectMember(ctx context.Context, find *ProjectMemberFind) (*ProjectMember, error)
	PatchProjectMember(ctx context.Context, patch *ProjectMemberPatch) (*ProjectMember, error)
	DeleteProjectMember(ctx context.Context, delete *ProjectMemberDelete) error
}
