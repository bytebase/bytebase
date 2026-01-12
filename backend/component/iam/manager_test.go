package iam

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCheck(t *testing.T) {
	testUser := &store.UserMessage{
		ID:    123,
		Email: "test@example.com",
	}

	rolePermissions := make(map[string]map[permission.Permission]bool)
	for _, role := range store.PredefinedRoles {
		rolePermissions[common.FormatRole(role.ResourceID)] = role.Permissions
	}
	getPermissions := func(role string) map[permission.Permission]bool {
		return rolePermissions[role]
	}

	tests := []struct {
		permission   permission.Permission
		policy       *storepb.IamPolicy
		groupMembers map[string]map[string]bool
		want         bool
	}{
		{
			permission: permission.InstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceMember",
						Members: []string{"users/test@example.com"},
					},
				},
			},
			groupMembers: nil,
			want:         false,
		},
		{
			permission: permission.InstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/test@example.com"},
					},
				},
			},
			groupMembers: nil,
			want:         true,
		},
		{
			permission: permission.InstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/other@example.com"},
					},
				},
			},
			groupMembers: nil,
			want:         false,
		},
		{
			permission: permission.InstancesCreate,
			policy: &storepb.IamPolicy{
				Bindings: []*storepb.Binding{
					{
						Role:    "roles/workspaceAdmin",
						Members: []string{"users/other@example.com", common.AllUsers},
					},
				},
			},
			groupMembers: nil,
			want:         true,
		},
		{
			permission: permission.InstancesCreate,
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
					"users/test@example.com": true,
				},
			},
			want: true,
		}}

	for i, test := range tests {
		getGroupMembers := func(groupName string) map[string]bool {
			if test.groupMembers == nil {
				return nil
			}
			return test.groupMembers[groupName]
		}
		got := check(testUser, test.permission, test.policy, getPermissions, getGroupMembers)
		if got != test.want {
			require.Equal(t, test.want, got, i)
		}
	}
}
