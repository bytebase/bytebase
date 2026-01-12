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
)

type Manager struct {
	store          *store.Store
	licenseService *enterprise.LicenseService
}

func NewManager(store *store.Store, licenseService *enterprise.LicenseService) (*Manager, error) {
	m := &Manager{
		store:          store,
		licenseService: licenseService,
	}
	return m, nil
}

// Check if the user has permission on the resource hierarchy.
// CEL on the binding is not considered.
// When multiple projects are specified, the user should have permission on every projects.
func (m *Manager) CheckPermission(ctx context.Context, p permission.Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	getPermissions := func(role string) map[permission.Permission]bool {
		perms, _ := m.GetPermissions(ctx, role)
		return perms
	}
	getGroupMembers := func(groupName string) map[string]bool {
		members, _ := m.store.GetGroupMembersSnapshot(ctx, groupName)
		return members
	}

	policyMessage, err := m.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, err
	}
	if ok := check(user, p, policyMessage.Policy, getPermissions, getGroupMembers); ok {
		return true, nil
	}

	if len(projectIDs) > 0 {
		allOK := true
		for _, projectID := range projectIDs {
			project, err := m.store.GetProject(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return false, err
			}
			if project == nil {
				return false, errors.Errorf("project %q not found", projectID)
			}
			policyMessage, err := m.store.GetProjectIamPolicy(ctx, project.ResourceID)
			if err != nil {
				return false, err
			}
			if ok := check(user, p, policyMessage.Policy, getPermissions, getGroupMembers); !ok {
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
func (m *Manager) GetPermissions(ctx context.Context, roleName string) (map[permission.Permission]bool, error) {
	resourceID := strings.TrimPrefix(roleName, "roles/")
	role, err := m.store.GetRoleSnapshot(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, nil
	}
	return maps.Clone(role.Permissions), nil
}

func (m *Manager) GetUserGroups(ctx context.Context, email string) ([]string, error) {
	return m.store.GetUserGroupsSnapshot(ctx, common.FormatUserEmail(email))
}

func check(user *store.UserMessage, p permission.Permission, policy *storepb.IamPolicy, getPermissions func(role string) map[permission.Permission]bool, getGroupMembers func(groupName string) map[string]bool) bool {
	userName := common.FormatUserEmail(user.Email)

	for _, binding := range policy.GetBindings() {
		permissions := getPermissions(binding.GetRole())
		if permissions == nil {
			continue
		}
		if !permissions[p] {
			continue
		}
		for _, member := range binding.GetMembers() {
			if member == common.AllUsers {
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
