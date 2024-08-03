package iam

import (
	"context"
	_ "embed"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

//go:embed acl.yaml
var aclYaml []byte

type acl struct {
	Roles []struct {
		Name        string   `yaml:"name"`
		Permissions []string `yaml:"permissions"`
	} `yaml:"roles"`
}

type Manager struct {
	predefinedRoles map[string]map[Permission]bool
	store           *store.Store
	licenseService  enterprise.LicenseService
	// user uid: workspace role list
	userRoleCache *lru.Cache[int, []string]
}

func NewManager(store *store.Store, licenseService enterprise.LicenseService) (*Manager, error) {
	predefinedACL := new(acl)
	if err := yaml.Unmarshal(aclYaml, predefinedACL); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal predefined acl")
	}

	userRoleCache, err := lru.New[int, []string](32768)
	if err != nil {
		return nil, err
	}

	predefinedRoles := make(map[string]map[Permission]bool)
	for _, binding := range predefinedACL.Roles {
		for _, permission := range binding.Permissions {
			if _, ok := predefinedRoles[binding.Name]; !ok {
				predefinedRoles[binding.Name] = make(map[Permission]bool)
			}
			predefinedRoles[binding.Name][Permission(permission)] = true
		}
	}

	return &Manager{
		predefinedRoles: predefinedRoles,
		store:           store,
		licenseService:  licenseService,
		userRoleCache:   userRoleCache,
	}, nil
}

// Check if the user or `allUsers` or the user group has the permission p
// or has the permission p in every project.
func (m *Manager) CheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	if m.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
		// nolint
		return true, nil
	}

	ok, err := m.doCheckPermission(ctx, p, user, projectIDs...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check permission")
	}
	if ok {
		return true, nil
	}
	allUsers, err := m.store.GetUserByID(ctx, api.AllUsersID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get allUsers")
	}
	return m.doCheckPermission(ctx, p, allUsers, projectIDs...)
}

func (m *Manager) ClearCache() {
	m.userRoleCache.Purge()
}

func (m *Manager) doCheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	workspaceRoles, err := m.getWorkspaceRoles(ctx, user)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get workspace roles")
	}
	projectRoles, err := m.getProjectRoles(ctx, user, projectIDs)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project roles")
	}
	return m.hasPermission(ctx, p, workspaceRoles, projectRoles)
}

// GetPermissions returns all permissions for the given role.
// Role format is roles/{role}.
func (m *Manager) GetPermissions(ctx context.Context, roleName string) (map[Permission]bool, error) {
	if permissions, ok := m.predefinedRoles[roleName]; ok {
		return permissions, nil
	}
	roleID, err := common.GetRoleID(roleName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get role id from %q", roleName)
	}
	role, err := m.store.GetRole(ctx, roleID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get role %q", roleID)
	}
	if role == nil {
		return nil, nil
	}
	return role.Permissions, nil
}

func (m *Manager) hasPermission(ctx context.Context, p Permission, workspaceRoles []string, projectRoles [][]string) (bool, error) {
	ok, err := m.hasPermissionOnWorkspace(ctx, p, workspaceRoles)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check permission on workspace")
	}
	if ok {
		return true, nil
	}
	ok, err = m.hasPermissionOnEveryProject(ctx, p, projectRoles)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check permission on every project")
	}
	return ok, nil
}

func (m *Manager) hasPermissionOnWorkspace(ctx context.Context, p Permission, workspaceRoles []string) (bool, error) {
	for _, role := range workspaceRoles {
		permissions, err := m.GetPermissions(ctx, role)
		if err != nil {
			return false, errors.Wrapf(err, "failed to get permissions")
		}
		if permissions[p] {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) hasPermissionOnEveryProject(ctx context.Context, p Permission, projectRoles [][]string) (bool, error) {
	if len(projectRoles) == 0 {
		return false, nil
	}
	for _, projectRole := range projectRoles {
		has := false
		for _, role := range projectRole {
			permissions, err := m.GetPermissions(ctx, role)
			if err != nil {
				return false, errors.Wrapf(err, "failed to get permissions")
			}
			if permissions[p] {
				has = true
				break
			}
		}
		if !has {
			return false, nil
		}
	}
	return true, nil
}

// getWorkspaceRoles returns roles for the user in the workspace IAM policy.
func (m *Manager) getWorkspaceRoles(ctx context.Context, user *store.UserMessage) ([]string, error) {
	if v, ok := m.userRoleCache.Get(user.ID); ok {
		return v, nil
	}
	policyMessage, err := m.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, err
	}
	roles := utils.GetUserRolesInIamPolicy(ctx, m.store, user, policyMessage.Policy)
	m.userRoleCache.Add(user.ID, roles)

	return roles, nil
}

func (m *Manager) getProjectRoles(ctx context.Context, user *store.UserMessage, projectIDs []string) ([][]string, error) {
	var roles [][]string
	for _, projectID := range projectIDs {
		project, err := m.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectID,
		})
		if err != nil {
			return nil, err
		}
		if project == nil {
			return nil, errors.Errorf("cannot found project %s", projectID)
		}

		policyMessage, err := m.store.GetProjectIamPolicy(ctx, project.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get iam policy for project %q", projectID)
		}

		projectRoles := utils.GetUserRolesInIamPolicy(ctx, m.store, user, policyMessage.Policy)
		roles = append(roles, projectRoles)
	}
	return roles, nil
}
