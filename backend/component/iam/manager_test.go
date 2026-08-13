package iam

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCheck(t *testing.T) {
	testUser := &store.UserMessage{
		ID:    123,
		Email: "test@example.com",
		Type:  storepb.PrincipalType_END_USER,
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
		got := check(testUser, test.permission, test.policy, getPermissions, getGroupMembers, false, false /* projectWideOnly */)
		if got != test.want {
			require.Equal(t, test.want, got, i)
		}
	}
}

func TestCheckProjectWideOnlySkipsResourceScopedBindings(t *testing.T) {
	testUser := &store.UserMessage{
		ID:    123,
		Email: "test@example.com",
		Type:  storepb.PrincipalType_END_USER,
	}

	rolePermissions := make(map[string]map[permission.Permission]bool)
	for _, role := range store.PredefinedRoles {
		rolePermissions[common.FormatRole(role.ResourceID)] = role.Permissions
	}
	getPermissions := func(role string) map[permission.Permission]bool {
		return rolePermissions[role]
	}
	getGroupMembers := func(string) map[string]bool { return nil }

	policyWithCondition := func(expression string) *storepb.IamPolicy {
		binding := &storepb.Binding{
			Role:    "roles/projectOwner",
			Members: []string{"users/test@example.com"},
		}
		if expression != "" {
			binding.Condition = &expr.Expr{Expression: expression}
		}
		return &storepb.IamPolicy{Bindings: []*storepb.Binding{binding}}
	}

	testCases := []struct {
		name       string
		expression string
		// A resource-scoped grant must confer nothing project-wide, while the
		// generic check keeps honoring it.
		wantProjectWide bool
		wantGeneric     bool
	}{
		{name: "no condition", expression: "", wantProjectWide: true, wantGeneric: true},
		{
			name:            "expiry only",
			expression:      `request.time < timestamp("2099-01-01T00:00:00Z")`,
			wantProjectWide: true,
			wantGeneric:     true,
		},
		{
			name:            "database scoped",
			expression:      `resource.database == "instances/i/databases/d"`,
			wantProjectWide: false,
			wantGeneric:     true,
		},
		{
			name:            "environment scoped with expiry",
			expression:      `resource.environment_id == "prod" && request.time < timestamp("2099-01-01T00:00:00Z")`,
			wantProjectWide: false,
			wantGeneric:     true,
		},
		{
			name:            "expired",
			expression:      `request.time < timestamp("2000-01-01T00:00:00Z")`,
			wantProjectWide: false,
			wantGeneric:     false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			policy := policyWithCondition(tc.expression)
			require.Equal(t, tc.wantProjectWide,
				check(testUser, permission.SavedQueriesManage, policy, getPermissions, getGroupMembers, false, true /* projectWideOnly */))
			require.Equal(t, tc.wantGeneric,
				check(testUser, permission.SavedQueriesManage, policy, getPermissions, getGroupMembers, false, false /* projectWideOnly */))
		})
	}
}
