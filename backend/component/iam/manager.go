package iam

import (
	"context"
	_ "embed"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
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
	predefinedRoles map[string][]Permission
	store           *store.Store
}

func NewManager(store *store.Store) (*Manager, error) {
	predefinedACL := new(acl)
	if err := yaml.Unmarshal(aclYaml, predefinedACL); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal predefined acl")
	}

	predefinedRoles := make(map[string][]Permission)
	for _, binding := range predefinedACL.Roles {
		for _, permission := range binding.Permissions {
			predefinedRoles[binding.Name] = append(predefinedRoles[binding.Name], NewPermission(permission))
		}
	}

	return &Manager{
		predefinedRoles: predefinedRoles,
		store:           store,
	}, nil
}

// Check if the user or `allUsers` has the permission p
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
func (m *Manager) GetPermissions(ctx context.Context, roleName string) ([]Permission, error) {
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
	var permissions []Permission
	for _, permission := range role.Permissions.GetPermissions() {
		permissions = append(permissions, NewPermission(permission))
	}
	return permissions, nil
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
		if slices.Contains(permissions, p) {
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
			if slices.Contains(permissions, p) {
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
		find := &store.GetProjectPolicyMessage{}
		projectUID, isNumber := isNumber(projectID)
		if isNumber {
			find.UID = &projectUID
		} else {
			projectID := projectID
			find.ProjectID = &projectID
		}
		iamPolicy, err := m.store.GetProjectPolicy(ctx, find)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get iam policy for project %q", projectID)
		}
		projectRoles := getRolesFromProjectPolicy(user, iamPolicy)
		roles = append(roles, projectRoles)
	}
	return roles, nil
}

func getRolesFromProjectPolicy(user *store.UserMessage, policy *store.IAMPolicyMessage) []string {
	var roles []string
	for _, binding := range policy.Bindings {
		ok, err := common.EvalBindingCondition(binding.Condition.GetExpression(), time.Now())
		if err != nil {
			slog.Error("failed to eval member condition", "expression", binding.Condition.GetExpression(), log.BBError(err))
			continue
		}
		if !ok {
			continue
		}
		for _, member := range binding.Members {
			if member.ID == user.ID || member.Email == api.AllUsers {
				roles = append(roles, common.FormatRole(binding.Role.String()))
				break
			}
		}
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
