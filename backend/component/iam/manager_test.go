package iam

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestCheck(t *testing.T) {
	roles, err := loadPredefinedRoles()
	require.NoError(t, err)
	rolePermissions := make(map[string]map[Permission]bool)
	for _, role := range roles {
		rolePermissions[common.FormatRole(role.ResourceID)] = role.Permissions
	}
	userID := 123

	tests := []struct {
		permission Permission
		policy     *storepb.IamPolicy
		want       bool
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
			want: false,
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
			want: true,
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
			want: false,
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
			want: true,
		},
	}

	for _, test := range tests {
		got := check(userID, test.permission, test.policy, rolePermissions)
		if got != test.want {
			require.Equal(t, test.want, got)
		}
	}
}
