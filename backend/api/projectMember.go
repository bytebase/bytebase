package api

import "context"

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
	ID string `jsonapi:"primary,project-member"`

	// Related fields
	Project *ResourceObject `jsonapi:"relation,project"`

	// Standard fields
	WorkspaceId int
	CreatorId   int   `jsonapi:"attr,creatorId"`
	CreatedTs   int64 `jsonapi:"attr,createdTs"`
	UpdaterId   int   `jsonapi:"attr,updaterId"`
	UpdatedTs   int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Role        string `jsonapi:"attr,role"`
	PrincipalId int    `jsonapi:"attr,principalId"`
}

type ProjectMemberCreate struct {
	// Related fields
	ProjectId int

	// Standard fields
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Role        ProjectRole `jsonapi:"attr,role"`
	PrincipalId int         `jsonapi:"attr,principalId"`
}

type ProjectMemberFind struct {
	// Related fields
	ProjectId *int `jsonapi:"attr,projectId"`

	// Standard fields
	ID          *int
	WorkspaceId *int
}

type ProjectMemberPatch struct {
	// Standard fields
	ID          int
	WorkspaceId int
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}

type ProjectMemberDelete struct {
	// Standard fields
	ID int
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type ProjectMemberService interface {
	CreateProjectMember(ctx context.Context, create *ProjectMemberCreate) (*ProjectMember, error)
	FindProjectMemberList(ctx context.Context, find *ProjectMemberFind) ([]*ProjectMember, error)
	PatchProjectMemberByID(ctx context.Context, patch *ProjectMemberPatch) (*ProjectMember, error)
	DeleteProjectMemberByID(ctx context.Context, delete *ProjectMemberDelete) error
}
