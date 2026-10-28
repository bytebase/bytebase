package store

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/permission"
)

func TestProjectOwnerInstancePermissions(t *testing.T) {
	t.Parallel()
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

func TestSQLEditorExporterRole(t *testing.T) {
	t.Parallel()
	exporter := GetPredefinedRole(SQLEditorExporterRole)
	require.NotNil(t, exporter)
	require.Equal(t, "SQL Editor Exporter", exporter.Name)

	want := maps.Clone(GetPredefinedRole(SQLEditorReadUserRole).Permissions)
	want[permission.SQLExport] = true
	require.Equal(t, want, exporter.Permissions)
	for p := range exporter.Permissions {
		require.Truef(t, permission.Exists(p), "permission %s is not defined in permission.yaml", p)
	}
}

func TestPredefinedRolesSQLExport(t *testing.T) {
	t.Parallel()
	for roleID, want := range map[string]bool{
		WorkspaceAdminRole:    true,
		WorkspaceDBARole:      true,
		ProjectOwnerRole:      true,
		SQLEditorExporterRole: true,
		SQLEditorUserRole:     false,
		SQLEditorReadUserRole: false,
	} {
		require.Equalf(t, want, GetPredefinedRole(roleID).Permissions[permission.SQLExport], "role %s", roleID)
	}
}

// Test that every permission in predefined roles is also defined in permission.yaml.
func TestPredefinedRolesPermissionsExist(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	for _, role := range PredefinedRoles {
		for p := range role.Permissions {
			exist := permission.Exists(p)
			a.True(exist, "permission %s is not defined in permission.yaml", p)
		}
	}
}
