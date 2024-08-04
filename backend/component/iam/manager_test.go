package iam

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestCheck(t *testing.T) {
	userID := 123

	roles, err := loadPredefinedRoles()
	require.NoError(t, err)
	rolePermissions := make(map[string]map[Permission]bool)
	for _, role := range roles {
		rolePermissions[common.FormatRole(role.ResourceID)] = role.Permissions
	}

	tests := []struct {
		permission       Permission
		policy           *storepb.IamPolicy
		userGroupMembers map[string]map[string]bool
		want             bool
	}{
		{
			permission: PermissionInstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceMember",
						Members: []string{"users/123"},
					},
				},
			},
			userGroupMembers: nil,
			want:             false,
		},
		{
			permission: PermissionInstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/123"},
					},
				},
			},
			userGroupMembers: nil,
			want:             true,
		},
		{
			permission: PermissionInstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/321"},
					},
				},
			},
			userGroupMembers: nil,
			want:             false,
		},
		{
			permission: PermissionInstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/321", api.AllUsers},
					},
				},
			},
			userGroupMembers: nil,
			want:             true,
		},
		{
			permission: PermissionInstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"groups/eng@bytebase.com"},
					},
				},
			},
			userGroupMembers: map[string]map[string]bool{
				"groups/eng@bytebase.com": {
					"users/123": true,
				},
			},
			want: true,
		}}

	for i, test := range tests {
		got := check(userID, test.permission, test.policy, rolePermissions, test.userGroupMembers)
		if got != test.want {
			require.Equal(t, test.want, got, i)
		}
	}
}
