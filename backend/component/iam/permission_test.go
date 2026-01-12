package iam

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store"
)

// Test that every permission in predefined roles is also defined in permission.yaml.
func TestPermissionExists(t *testing.T) {
	a := require.New(t)

	for _, role := range store.PredefinedRoles {
		for p := range role.Permissions {
			exist := PermissionsExist(p)
			a.True(exist, "permission %s is not defined in permission.yaml", p)
		}
	}
}
