package iam

import (
	"testing"

	"github.com/stretchr/testify/require"
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
