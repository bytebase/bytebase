package iam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that every permission in the yaml is also defined in permission.go as a constant.
func TestPermissionExists(t *testing.T) {
	a := require.New(t)

	m, err := NewManager()
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
	case PermissionInstanceList:
		return true
	case PermissionInstanceGet:
		return true
	case PermissionInstanceCreate:
		return true
	case PermissionInstanceUpdate:
		return true
	case PermissionInstanceDelete:
		return true
	case PermissionInstanceUndelete:
		return true
	case PermissionInstanceSync:
		return true
	default:
		return false
	}
}
