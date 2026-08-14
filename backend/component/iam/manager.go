package iam

import (
	"context"
	"log/slog"
	"maps"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
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
	return m.doCheckPermission(ctx, p, user, workspaceID, false /* projectWideOnly */, projectIDs...)
}

// CheckProjectWidePermission is CheckPermission for permissions that a
// resource-scoped grant must not confer. A binding whose condition narrows to
// specific databases, schemas, tables, or environments is skipped entirely,
// so a data-slice grant cannot widen into a project-wide surface. Expiry is
// unaffected: ValidateIAMBinding already drops bindings whose time window has
// closed.
func (m *Manager) CheckProjectWidePermission(ctx context.Context, p permission.Permission, user *store.UserMessage, workspaceID string, projectIDs ...string) (bool, error) {
	return m.doCheckPermission(ctx, p, user, workspaceID, true /* projectWideOnly */, projectIDs...)
}

func (m *Manager) doCheckPermission(ctx context.Context, p permission.Permission, user *store.UserMessage, workspaceID string, projectWideOnly bool, projectIDs ...string) (bool, error) {
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
	if ok := check(user, p, policyMessage.Policy, getPermissions, getGroupMembers, m.saas, projectWideOnly); ok {
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
			if ok := check(user, p, policyMessage.Policy, getPermissions, getGroupMembers, false, projectWideOnly); !ok {
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

// PrincipalMembers returns the policy members that stand for a caller: itself,
// in the form its principal type dictates, plus each group it belongs to.
//
// This is the derivation check() uses, exposed for policy surfaces that
// evaluate their own bindings instead of going through CheckPermission. Taking
// the caller's type from formatUserNameByType is what keeps a service account
// out of a "users/" member, so a binding naming one there matches nobody --
// which is why a policy's write path can validate members by prefix alone.
// Groups are named by reference and never expanded.
func (m *Manager) PrincipalMembers(ctx context.Context, workspaceID string, user *store.UserMessage) ([]string, error) {
	groups, err := m.GetUserGroups(ctx, workspaceID, user.Email)
	if err != nil {
		return nil, err
	}
	return append([]string{formatUserNameByType(user)}, groups...), nil
}

func check(user *store.UserMessage, p permission.Permission, policy *storepb.IamPolicy, getPermissions func(role string) map[permission.Permission]bool, getGroupMembers func(groupName string) map[string]bool, skipAllUsers bool, projectWideOnly bool) bool {
	userName := formatUserNameByType(user)

	for _, binding := range policy.GetBindings() {
		if !utils.ValidateIAMBinding(binding) {
			continue
		}
		if projectWideOnly {
			scoped, err := conditionScopesResources(binding.Condition.GetExpression())
			if err != nil {
				// An unparsable condition is not a grant.
				slog.Error("failed to inspect binding condition", slog.String("expression", binding.Condition.GetExpression()), log.BBError(err))
				continue
			}
			if scoped {
				continue
			}
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
