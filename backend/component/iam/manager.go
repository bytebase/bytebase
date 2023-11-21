package iam

import (
	"context"
	_ "embed"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

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

func (m *Manager) CheckPermission(_ context.Context, p Permission, user *store.UserMessage) (bool, error) {
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
