package iam

import (
	"context"
	_ "embed"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

//go:embed acl.yaml
var aclYaml []byte

type acl struct {
	Roles []Role `yaml:"roles"`
}

type Role struct {
	Name        string   `yaml:"name"`
	Permissions []string `yaml:"permissions"`
}

type Manager struct {
	roles          map[string][]Permission
	store          *store.Store
	licenseService enterprise.LicenseService
}

func NewManager(store *store.Store, licenseService enterprise.LicenseService) (*Manager, error) {
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
		roles:          roles,
		store:          store,
		licenseService: licenseService,
	}, nil
}

func (m *Manager) getUser(ctx context.Context) (*store.UserMessage, error) {
	principalPtr := ctx.Value(common.PrincipalIDContextKey)
	if principalPtr == nil {
		return nil, errors.Errorf("principal ID not found")
	}
	principalID, ok := principalPtr.(int)
	if !ok {
		return nil, errors.Errorf("principal ID not found")
	}
	user, err := m.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find user %d", principalID)
	}
	if user == nil {
		return nil, errors.Errorf("user %d not found", principalID)
	}
	if user.MemberDeleted {
		return nil, errors.Errorf("user %d has been deactivated", principalID)
	}

	// If RBAC feature is not enabled, all users are treated as OWNER.
	if m.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
		user.Role = api.Owner
	}
	return user, nil
}

func (m *Manager) getUserRoles(user *store.UserMessage) []string {
	var roles []string
	switch user.Role {
	case "OWNER":
		roles = append(roles, "roles/workspaceAdmin")
	case "DBA":
		roles = append(roles, "roles/workspaceDBA")
	case "DEVELOPER":
		roles = append(roles, "roles/workspaceMember")
	}
	return roles
}

func (m *Manager) CheckPermission(ctx context.Context, p Permission) (bool, error) {
	user, err := m.getUser(ctx)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get user")
	}
	roles := m.getUserRoles(user)
	return m.hasPermission(p, roles...), nil
}

func (m *Manager) hasPermission(p Permission, roles ...string) bool {
	for _, role := range roles {
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
