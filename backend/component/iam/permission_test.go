package iam

import (
	"slices"
	"testing"

	_ "embed"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test that every permission in the yaml is also defined in permission.go as a constant.
func TestPermissionExists(t *testing.T) {
	a := require.New(t)

	predefinedRoles, err := loadPredefinedRoles()
	a.NoError(err)

	for _, role := range predefinedRoles {
		for p := range role.Permissions {
			exist := PermissionsExist(p)
			a.True(exist, "permission %s is not defined as a constant", p)
		}
	}
}

//go:embed permission.yaml
var permissionYaml []byte

// Test that permissions are equal in permission.yaml and allPermissions in permission.go.
func TestPermissionEquals(t *testing.T) {
	a := require.New(t)

	var permissions struct {
		Permissions []string `yaml:"permissions"`
	}

	a.NoError(yaml.Unmarshal(permissionYaml, &permissions))

	slices.Sort(permissions.Permissions)
	slices.Sort(allPermissions)

	a.Equal(permissions.Permissions, allPermissions)
}
