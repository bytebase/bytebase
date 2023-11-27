package iam

import (
	"context"
	_ "embed"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

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
	roles map[string][]Permission
	store *store.Store
}

func NewManager(store *store.Store) (*Manager, error) {
	predefinedACL := new(acl)
	if err := yaml.Unmarshal(aclYaml, predefinedACL); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal predefined acl")
	}

	roles := make(map[string][]Permission)
	for _, binding := range predefinedACL.Roles {
		for _, permission := range binding.Permissions {
			roles[binding.Name] = append(roles[binding.Name], Permission(permission))
		}
	}

	return &Manager{
		roles: roles,
		store: store,
	}, nil
}

// Check if the user has the permission p
// or has the permission p in every project.
func (m *Manager) CheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	workspaceRoles, err := m.getWorkspaceRoles(user)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get workspace roles")
	}
	projectRoles, err := m.getProjectRoles(ctx, user, projectIDs)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project roles")
	}

	return m.hasPermission(p, workspaceRoles, projectRoles), nil
}

func (m *Manager) hasPermission(p Permission, workspaceRoles []string, projectRoles [][]string) bool {
	return m.hasPermissionOnWorkspace(p, workspaceRoles) ||
		m.hasPermissionOnEveryProject(p, projectRoles)
}

func (m *Manager) hasPermissionOnWorkspace(p Permission, workspaceRoles []string) bool {
	for _, role := range workspaceRoles {
		permissions, ok := m.roles[role]
		if !ok {
			continue
		}
		for _, permission := range permissions {
			if permission == p {
				return true
			}
		}
	}
	return false
}

func (m *Manager) hasPermissionOnEveryProject(p Permission, projectRoles [][]string) bool {
	for _, projectRole := range projectRoles {
		has := false
		for _, role := range projectRole {
			permissions, ok := m.roles[role]
			if !ok {
				continue
			}
			for _, permission := range permissions {
				if permission == p {
					has = true
					break
				}
			}
		}
		if !has {
			return false
		}
	}
	return true
}
func (*Manager) getWorkspaceRoles(user *store.UserMessage) ([]string, error) {
	role, err := convertWorkspaceRole(user.Role.String())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace roles")
	}
	return []string{role}, nil
}

func (m *Manager) getProjectRoles(ctx context.Context, user *store.UserMessage, projectIDs []string) ([][]string, error) {
	var roles [][]string
	for _, projectID := range projectIDs {
		iamPolicy, err := m.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &projectID})
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
		// TODO(p0ny): eval binding.Condition
		for _, member := range binding.Members {
			if member.ID == user.ID || member.Email == api.AllUsers {
				roles = append(roles, convertProjectRole(binding.Role.String()))
				break
			}
		}
	}
	return roles
}

func convertProjectRole(role string) string {
	switch role {
	case "OWNER":
		return "roles/projectOwner"
	case "DEVELOPER":
		return "roles/projectDeveloper"
	case "QUERIER":
		return "roles/projectQuerier"
	case "EXPORTER":
		return "roles/projectExporter"
	case "RELEASER":
		return "roles/projectReleaser"
	case "VIEWER":
		return "roles/projectViewer"
	default:
		return "roles/" + role
	}
}

func convertWorkspaceRole(role string) (string, error) {
	switch role {
	case "OWNER":
		return "roles/workspaceAdmin", nil
	case "DBA":
		return "roles/workspaceDBA", nil
	case "DEVELOPER":
		return "roles/workspaceMember", nil
	default:
		return "", errors.Errorf("unexpected workspace role %q", role)
	}
}
