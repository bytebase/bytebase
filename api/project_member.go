package api

import (
	"github.com/bytebase/bytebase/common"
)

// ProjectMember is the API message for project members.
type ProjectMember struct {
	ID int `jsonapi:"primary,projectMember"`

	// Related fields
	// Just returns ProjectID otherwise would cause circular dependency.
	ProjectID int `jsonapi:"attr,projectId"`

	// Domain specific fields
	Role      string     `jsonapi:"attr,role"`
	Principal *Principal `jsonapi:"relation,principal"`
}

// ProjectMemberCreate is the API message for creating a project member.
type ProjectMemberCreate struct {
	Role        common.ProjectRole `jsonapi:"attr,role"`
	PrincipalID int                `jsonapi:"attr,principalId"`
}

// ProjectMemberPatch is the API message for patching a project member.
type ProjectMemberPatch struct {
	Role *string `jsonapi:"attr,role"`
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
	// ProjectPermissionSyncSheet allows user to sync sheet for the project with VCS configured.
	ProjectPermissionSyncSheet ProjectPermissionType = "bb.permission.project.sync-sheet"
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
		ProjectPermissionSyncSheet:      {true, true},
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
