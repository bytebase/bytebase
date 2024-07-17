package iam

import (
	"context"
	_ "embed"
	"strconv"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
}

func NewManager(store *store.Store) (*Manager, error) {
	predefinedACL := new(acl)
	if err := yaml.Unmarshal(aclYaml, predefinedACL); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal predefined acl")
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
	}, nil
}

// Check if the user or `allUsers` or the user group has the permission p
// or has the permission p in every project.
func (m *Manager) CheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
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

func (m *Manager) doCheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	workspaceRoles := m.getWorkspaceRoles(user)
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
	if GetPermissionLevel(p) == PermissionLevelWorkspace {
		return false, nil
	}
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

func (*Manager) getWorkspaceRoles(user *store.UserMessage) []string {
	var roles []string
	for _, r := range user.Roles {
		roles = append(roles, common.FormatRole(r.String()))
	}
	return roles
}

func (m *Manager) getProjectRoles(ctx context.Context, user *store.UserMessage, projectIDs []string) ([][]string, error) {
	var roles [][]string
	for _, projectID := range projectIDs {
		projectUID, isNumber := isNumber(projectID)

		if !isNumber {
			project, err := m.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID: &projectID,
			})
			if err != nil {
				return nil, err
			}
			if project == nil {
				return nil, errors.Errorf("cannot found project %s", projectID)
			}
			projectUID = project.UID
		}

		iamPolicy, err := m.store.GetProjectIamPolicy(ctx, projectUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get iam policy for project %q", projectID)
		}

		projectRoles := getRolesFromProjectPolicy(ctx, m.store, user, iamPolicy)
		roles = append(roles, projectRoles)
	}
	return roles, nil
}

func getRolesFromProjectPolicy(ctx context.Context, stores *store.Store, user *store.UserMessage, policy *storepb.IamPolicy) []string {
	var roles []string
	bindings := utils.GetUserIAMPolicyBindings(ctx, stores, user, policy)

	for _, binding := range bindings {
		roles = append(roles, binding.Role)
	}
	return roles
}

func isNumber(v string) (int, bool) {
	n, err := strconv.Atoi(v)
	if err == nil {
		return int(n), true
	}
	return 0, false
}
