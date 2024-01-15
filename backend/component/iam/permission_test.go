package iam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that every permission in the yaml is also defined in permission.go as a constant.
func TestPermissionExists(t *testing.T) {
	a := require.New(t)

	m, err := NewManager(nil)
	a.NoError(err)

	for _, permissions := range m.roles {
		for _, p := range permissions {
			exist := permissionExist(p)
			a.True(exist, "permission %s is not defined as a constant", p)
		}
	}
}

func permissionExist(p Permission) bool {
	//exhaustive:enforce
	switch p {
	case
		PermissionBackupsCreate,
		PermissionBackupsList,
		PermissionBranchesCreate,
		PermissionBranchesDelete,
		PermissionBranchesGet,
		PermissionBranchesList,
		PermissionBranchesUpdate,
		PermissionChangeHistoriesGet,
		PermissionChangeHistoriesList,
		PermissionChangelistsCreate,
		PermissionChangelistsDelete,
		PermissionChangelistsGet,
		PermissionChangelistsList,
		PermissionChangelistsUpdate,
		PermissionDatabaseSecretsDelete,
		PermissionDatabaseSecretsList,
		PermissionDatabaseSecretsUpdate,
		PermissionDatabasesAdviseIndex,
		PermissionDatabasesExport,
		PermissionDatabasesGet,
		PermissionDatabasesGetBackupSetting,
		PermissionDatabasesGetSchema,
		PermissionDatabasesList,
		PermissionDatabasesQuery,
		PermissionDatabasesSync,
		PermissionDatabasesUpdate,
		PermissionDatabasesUpdateBackupSetting,
		PermissionEnvironmentsCreate,
		PermissionEnvironmentsDelete,
		PermissionEnvironmentsGet,
		PermissionEnvironmentsList,
		PermissionEnvironmentsUndelete,
		PermissionEnvironmentsUpdate,
		PermissionExternalVersionControlsCreate,
		PermissionExternalVersionControlsDelete,
		PermissionExternalVersionControlsGet,
		PermissionExternalVersionControlsList,
		PermissionExternalVersionControlsListProjects,
		PermissionExternalVersionControlsSearchProjects,
		PermissionExternalVersionControlsUpdate,
		PermissionIdentityProvidersCreate,
		PermissionIdentityProvidersDelete,
		PermissionIdentityProvidersGet,
		PermissionIdentityProvidersUndelete,
		PermissionIdentityProvidersUpdate,
		PermissionInstanceRolesCreate,
		PermissionInstanceRolesDelete,
		PermissionInstanceRolesGet,
		PermissionInstanceRolesList,
		PermissionInstanceRolesUndelete,
		PermissionInstanceRolesUpdate,
		PermissionInstancesCreate,
		PermissionInstancesDelete,
		PermissionInstancesGet,
		PermissionInstancesList,
		PermissionInstancesSync,
		PermissionInstancesUndelete,
		PermissionInstancesUpdate,
		PermissionInstancesAdminExecute,
		PermissionIssueCommentsCreate,
		PermissionIssueCommentsUpdate,
		PermissionIssuesCreate,
		PermissionIssuesGet,
		PermissionIssuesList,
		PermissionIssuesUpdate,
		PermissionPlanCheckRunsList,
		PermissionPlanCheckRunsRun,
		PermissionPlansCreate,
		PermissionPlansGet,
		PermissionPlansList,
		PermissionPlansUpdate,
		PermissionPoliciesCreate,
		PermissionPoliciesDelete,
		PermissionPoliciesGet,
		PermissionPoliciesList,
		PermissionPoliciesUpdate,
		PermissionProjectsCreate,
		PermissionProjectsDelete,
		PermissionProjectsGet,
		PermissionProjectsGetIAMPolicy,
		PermissionProjectsList,
		PermissionProjectsSetIAMPolicy,
		PermissionProjectsUndelete,
		PermissionProjectsUpdate,
		PermissionRisksCreate,
		PermissionRisksDelete,
		PermissionRisksList,
		PermissionRisksUpdate,
		PermissionRolesCreate,
		PermissionRolesDelete,
		PermissionRolesList,
		PermissionRolesUpdate,
		PermissionRolloutsCreate,
		PermissionRolloutsGet,
		PermissionRolloutsPreview,
		PermissionSettingsGet,
		PermissionSettingsList,
		PermissionSettingsSet,
		PermissionSlowQueriesList,
		PermissionTaskRunsCancel,
		PermissionTaskRunsList,
		PermissionTasksRun,
		PermissionTasksSkip:
		return true
	default:
		return false
	}
}
