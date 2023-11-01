package api

import (
	"github.com/bytebase/bytebase/backend/common"
)

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
func ProjectPermission(permission ProjectPermissionType, plan PlanType, roles map[common.ProjectRole]bool) bool {
	// a map from the a particular feature to the respective enablement of a project developer and owner.
	projectPermissionMatrix := map[ProjectPermissionType][3]bool{
		ProjectPermissionManageGeneral:  {false, true, true},
		ProjectPermissionManageMember:   {false, true, false},
		ProjectPermissionCreateSheet:    {true, true, false},
		ProjectPermissionAdminSheet:     {false, true, false},
		ProjectPermissionOrganizeSheet:  {true, true, false},
		ProjectPermissionSyncSheet:      {true, true, false},
		ProjectPermissionChangeDatabase: {true, true, false},
		ProjectPermissionAdminDatabase:  {false, true, false},
		// If dba-workflow is disabled, then project developer can also create database.
		ProjectPermissionCreateDatabase: {!Feature(FeatureDBAWorkflow, plan), true, false},
		// If dba-workflow is disabled, then project developer can also transfer database.
		ProjectPermissionTransferDatabase: {!Feature(FeatureDBAWorkflow, plan), true, false},
	}

	for role := range roles {
		switch role {
		case common.ProjectDeveloper:
			if projectPermissionMatrix[permission][0] {
				return true
			}
		case common.ProjectOwner:
			if projectPermissionMatrix[permission][1] {
				return true
			}
		case common.ProjectViewer:
			if projectPermissionMatrix[permission][2] {
				return true
			}
		}
	}

	return false
}
