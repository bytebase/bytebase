package api

import (
	"encoding/json"

	"github.com/bytebase/bytebase/common"
)

// ProjectRoleProvider is the role provider for a user in projects.
type ProjectRoleProvider string

const (
	// ProjectRoleProviderBytebase indicates the role provider is the Bytebase.
	ProjectRoleProviderBytebase ProjectRoleProvider = "BYTEBASE"
	// ProjectRoleProviderGitLabSelfHost indicates the role provider is the GitLab
	// self-hosted.
	ProjectRoleProviderGitLabSelfHost ProjectRoleProvider = "GITLAB_SELF_HOST"
	// ProjectRoleProviderGitHubCom indicates the role provider is the GitHub.com.
	ProjectRoleProviderGitHubCom ProjectRoleProvider = "GITHUB_COM"
)

// ProjectRoleProviderPayload is the payload for role provider.
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
	PrincipalID  *int
	Role         *Role
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
	ID        int
	ProjectID int

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

// ProjectPermissionType is the type of a project permission.
type ProjectPermissionType string

const (
	// ProjectPermissionManageGeneral allows user to manage general project settings.
	ProjectPermissionManageGeneral ProjectPermissionType = "bb.permission.project.manage-general"
	// ProjectPermissionManageMember allows user to manage project memberships.
	ProjectPermissionManageMember ProjectPermissionType = "bb.permission.project.manage-member"
	// ProjectPermissionCreateSheet allows user to create sheets in the project.
	ProjectPermissionCreateSheet ProjectPermissionType = "bb.permission.project.create-sheet"
	// ProjectPermissionAdminSheet allows user to manage sheet settings in the project.
	ProjectPermissionAdminSheet ProjectPermissionType = "bb.permission.project.admin-sheet"
	// ProjectPermissionOrganizeSheet allows user to organize sheet (star, pin) in the project.
	ProjectPermissionOrganizeSheet ProjectPermissionType = "bb.permission.project.organize-sheet"
	// ProjectPermissionChangeDatabase allows user to make DML/DDL database change in the project.
	ProjectPermissionChangeDatabase ProjectPermissionType = "bb.permission.project.change-database"
	// ProjectPermissionAdminDatabase allows user to manage database settings in the project.
	// - Edit database label.
	// - Backup settings.
	ProjectPermissionAdminDatabase ProjectPermissionType = "bb.permission.project.admin-database"
	// ProjectPermissionCreateDatabase allows user to create database in the project.
	ProjectPermissionCreateDatabase ProjectPermissionType = "bb.permission.project.create-database"
	// ProjectPermissionTransferDatabase allows user to transfer database out of/into the project.
	ProjectPermissionTransferDatabase ProjectPermissionType = "bb.permission.project.transfer-database"
)

// ProjectPermission returns whether a particular permission is granted to a particular project role in a particular plan.
func ProjectPermission(permission ProjectPermissionType, plan PlanType, role common.ProjectRole) bool {
	// a map from the a particular feature to the respective enablement of a project developer and owner.
	projectPermissionMatrix := map[ProjectPermissionType][2]bool{
		ProjectPermissionManageGeneral:  {false, true},
		ProjectPermissionManageMember:   {false, true},
		ProjectPermissionCreateSheet:    {true, true},
		ProjectPermissionAdminSheet:     {false, true},
		ProjectPermissionOrganizeSheet:  {true, true},
		ProjectPermissionChangeDatabase: {true, true},
		ProjectPermissionAdminDatabase:  {false, true},
		// If dba-workflow is disabled, then project developer can also create database.
		ProjectPermissionCreateDatabase: {!Feature(FeatureDBAWorkflow, plan), true},
		// If dba-workflow is disabled, then project developer can also transfer database.
		ProjectPermissionTransferDatabase: {!Feature(FeatureDBAWorkflow, plan), true},
	}

	switch role {
	case common.ProjectDeveloper:
		return projectPermissionMatrix[permission][0]
	case common.ProjectOwner:
		return projectPermissionMatrix[permission][1]
	}
	return false
}
