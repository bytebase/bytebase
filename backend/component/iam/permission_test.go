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
			exist := PermissionExist(p)
			a.True(exist, "permission %s is not defined as a constant", p)
		}
	}
}

func TestGetPermissionLevels(t *testing.T) {
	a := require.New(t)

	m, err := NewManager(nil)
	a.NoError(err)

	for _, permissions := range m.roles {
		for _, p := range permissions {
			level := GetPermissionLevel(p)
			a.NotEqual("", level.String(), "permission %s has no level defined", p)
		}
	}
}
