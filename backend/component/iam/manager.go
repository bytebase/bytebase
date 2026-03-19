package iam

import (
	"context"
	"maps"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

type Manager struct {
	store          *store.Store
	licenseService *enterprise.LicenseService
	saas           bool
}

func NewManager(store *store.Store, licenseService *enterprise.LicenseService, saas bool) (*Manager, error) {
	m := &Manager{
		store:          store,
		licenseService: licenseService,
		saas:           saas,
	}
	return m, nil
}

// Check if the user has permission on the resource hierarchy.
// CEL on the binding is not considered.
// When multiple projects are specified, the user should have permission on every projects.
func (m *Manager) CheckPermission(ctx context.Context, p permission.Permission, user *store.UserMessage, workspaceID string, projectIDs ...string) (bool, error) {
	getPermissions := func(role string) map[permission.Permission]bool {
		perms, _ := m.GetPermissions(ctx, workspaceID, role)
		return perms
	}
	getGroupMembers := func(groupName string) map[string]bool {
		members, _ := m.store.GetGroupMembersSnapshot(ctx, workspaceID, groupName)
		return members
	}

	policyMessage, err := m.store.GetWorkspaceIamPolicySnapshot(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	// In SaaS mode, skip allUsers for workspace-level IAM (members must be explicit).
	if ok := check(user, p, policyMessage.Policy, getPermissions, getGroupMembers, m.saas); ok {
		return true, nil
	}

	if len(projectIDs) > 0 {
		allOK := true
		for _, projectID := range projectIDs {
			project, err := m.store.GetProject(ctx, &store.FindProjectMessage{
				Workspace:   workspaceID,
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return false, err
			}
			if project == nil {
				return false, errors.Errorf("project %q not found", projectID)
			}
			policyMessage, err := m.store.GetProjectIamPolicySnapshot(ctx, workspaceID, project.ResourceID)
			if err != nil {
				return false, err
			}
			// Project-level: allUsers means "all workspace members", which is safe.
			if ok := check(user, p, policyMessage.Policy, getPermissions, getGroupMembers, false); !ok {
				allOK = false
				break
			}
		}
		return allOK, nil
	}
	return false, nil
}

func (m *Manager) ReloadCache(_ context.Context) error {
	m.store.PurgeGroupCaches()
	return nil
}

// GetPermissions returns all permissions for the given role.
// Role format is roles/{role}.
func (m *Manager) GetPermissions(ctx context.Context, workspaceID string, roleName string) (map[permission.Permission]bool, error) {
	resourceID := strings.TrimPrefix(roleName, "roles/")
	role, err := m.store.GetRoleSnapshot(ctx, workspaceID, resourceID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, nil
	}
	return maps.Clone(role.Permissions), nil
}

func (m *Manager) GetUserGroups(ctx context.Context, workspaceID string, email string) ([]string, error) {
	return m.store.GetUserGroupsSnapshot(ctx, workspaceID, common.FormatUserEmail(email))
}

func check(user *store.UserMessage, p permission.Permission, policy *storepb.IamPolicy, getPermissions func(role string) map[permission.Permission]bool, getGroupMembers func(groupName string) map[string]bool, skipAllUsers bool) bool {
	userName := formatUserNameByType(user)

	for _, binding := range policy.GetBindings() {
		if !utils.ValidateIAMBinding(binding) {
			continue
		}
		permissions := getPermissions(binding.GetRole())
		if permissions == nil {
			continue
		}
		if !permissions[p] {
			continue
		}
		for _, member := range binding.GetMembers() {
			if member == common.AllUsers && !skipAllUsers {
				return true
			}
			if member == userName {
				return true
			}
			if strings.HasPrefix(member, common.GroupPrefix) {
				if members := getGroupMembers(member); members != nil {
					if members[userName] {
						return true
					}
				}
			}
		}
	}
	return false
}

// formatUserNameByType returns the appropriate member name format based on user type.
// For regular users: users/{email}
// For service accounts: serviceAccounts/{email}
// For workload identities: workloadIdentities/{email}
func formatUserNameByType(user *store.UserMessage) string {
	switch user.Type {
	case storepb.PrincipalType_SERVICE_ACCOUNT:
		return common.FormatServiceAccountEmail(user.Email)
	case storepb.PrincipalType_WORKLOAD_IDENTITY:
		return common.FormatWorkloadIdentityEmail(user.Email)
	default:
		return common.FormatUserEmail(user.Email)
	}
}
