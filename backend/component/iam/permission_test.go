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
		PermissionInstancesList,
		PermissionInstancesGet,
		PermissionInstancesCreate,
		PermissionInstancesUpdate,
		PermissionInstancesDelete,
		PermissionInstancesUndelete,
		PermissionInstancesSync,
		PermissionDatabasesList,
		PermissionDatabasesGet,
		PermissionDatabasesUpdate,
		PermissionDatabasesSync,
		PermissionDatabasesGetMetadata,
		PermissionDatabasesUpdateMetadata,
		PermissionDatabasesGetSchema,
		PermissionDatabasesGetBackupSetting,
		PermissionDatabasesUpdateBackupSetting,
		PermissionBackupsList,
		PermissionBackupsCreate,
		PermissionChangeHistoriesList,
		PermissionChangeHistoriesGet,
		PermissionDatabaseSecretsList,
		PermissionDatabaseSecretsUpdate,
		PermissionDatabaseSecretsDelete,
		PermissionSlowQueriesList,
		PermissionEnvironmentsList,
		PermissionEnvironmentsGet,
		PermissionEnvironmentsCreate,
		PermissionEnvironmentsUpdate,
		PermissionEnvironmentsDelete,
		PermissionEnvironmentsUndelete,
		PermissionIssuesList,
		PermissionIssuesGet,
		PermissionIssuesCreate,
		PermissionIssuesUpdate,
		PermissionIssuesApprove,
		PermissionIssuesReject,
		PermissionIssueRerequestReview,
		PermissionIssueCommentsCreate,
		PermissionIssueCommentsUpdate:
		return true
	default:
		return false
	}
}
