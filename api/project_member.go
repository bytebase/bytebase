package api

import (
	"encoding/json"

	"github.com/bytebase/bytebase/common"
)

// ProjectRoleProvider is the role provider for a user in projects.
type ProjectRoleProvider string

const (
	// ProjectRoleProviderBytebase is the role provider of a project.
	ProjectRoleProviderBytebase ProjectRoleProvider = "BYTEBASE"
	// ProjectRoleProviderGitLabSelfHost is the role provider of a project.
	ProjectRoleProviderGitLabSelfHost ProjectRoleProvider = "GITLAB_SELF_HOST"
)

func (e ProjectRoleProvider) String() string {
	switch e {
	case ProjectRoleProviderBytebase:
		return "BYTEBASE"
	case ProjectRoleProviderGitLabSelfHost:
		return "GITLAB_SELF_HOST"
	}
	return ""
}

//ProjectRoleProviderPayload is the payload for role provider
type ProjectRoleProviderPayload struct {
	VCSRole    string `json:"vcsRole"`
	LastSyncTs int64  `json:"lastSyncTs"`
}

// ProjectMember is the API message for project members.
type ProjectMember struct {
	ID int `jsonapi:"primary,projectMember"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns ProjectID otherwise would cause circular dependency.
	ProjectID int `jsonapi:"attr,projectId"`

	// Domain specific fields
	Role         string `jsonapi:"attr,role"`
	PrincipalID  int
	Principal    *Principal          `jsonapi:"relation,principal"`
	RoleProvider ProjectRoleProvider `jsonapi:"attr,roleProvider"`
	Payload      string              `jsonapi:"attr,payload"`
}

// ProjectMemberCreate is the API message for creating a project member.
type ProjectMemberCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID int

	// Domain specific fields
	Role         common.ProjectRole  `jsonapi:"attr,role"`
	PrincipalID  int                 `jsonapi:"attr,principalId"`
	RoleProvider ProjectRoleProvider `jsonapi:"attr,roleProvider"`
	Payload      string              `jsonapi:"attr,payload"`
}

// ProjectMemberFind is the API message for finding project members.
type ProjectMemberFind struct {
	ID *int

	// Related fields
	ProjectID    *int
	RoleProvider *ProjectRoleProvider
}

func (find *ProjectMemberFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// ProjectMemberPatch is the API message for patching a project member.
type ProjectMemberPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Role         *string `jsonapi:"attr,role"`
	RoleProvider *string `jsonapi:"attr,roleProvider"`
	Payload      *string `jsonapi:"attr,payload"`
}

// ProjectMemberDelete is the API message for deleting a project member.
type ProjectMemberDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

// ProjectMemberBatchUpdate is the API message for batch updating project member.
type ProjectMemberBatchUpdate struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// All the Member to be update should have the same role provider as this field
	RoleProvider ProjectRoleProvider
	List         []*ProjectMemberCreate
}
