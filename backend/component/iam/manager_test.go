package iam

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
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
		permission   Permission
		policy       *storepb.IamPolicy
		groupMembers map[string]map[string]bool
		want         bool
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
			groupMembers: nil,
			want:         false,
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
			groupMembers: nil,
			want:         true,
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
			groupMembers: nil,
			want:         false,
		},
		{
			permission: PermissionInstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/321", base.AllUsers},
					},
				},
			},
			groupMembers: nil,
			want:         true,
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
			groupMembers: map[string]map[string]bool{
				"groups/eng@bytebase.com": {
					"users/123": true,
				},
			},
			want: true,
		}}

	for i, test := range tests {
		got := check(userID, test.permission, test.policy, rolePermissions, test.groupMembers)
		if got != test.want {
			require.Equal(t, test.want, got, i)
		}
	}
}
