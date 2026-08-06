package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/permission"
)

func TestProjectOwnerInstancePermissions(t *testing.T) {
	role := GetPredefinedRole(ProjectOwnerRole)
	require.NotNil(t, role)

	instancePermissions := []permission.Permission{
		permission.InstancesCreate,
		permission.InstancesDelete,
		permission.InstancesGet,
		permission.InstancesList,
		permission.InstancesSync,
		permission.InstancesUndelete,
		permission.InstancesUpdate,
	}
	for _, p := range instancePermissions {
		require.Truef(t, role.Permissions[p], "Project Owner must have %q", p)
	}
}

// Test that every permission in predefined roles is also defined in permission.yaml.
func TestPredefinedRolesPermissionsExist(t *testing.T) {
	a := require.New(t)

	for _, role := range PredefinedRoles {
		for p := range role.Permissions {
			exist := permission.Exists(p)
			a.True(exist, "permission %s is not defined in permission.yaml", p)
		}
	}
}
