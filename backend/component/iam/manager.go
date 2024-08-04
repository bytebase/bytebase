package iam

import (
	"context"
	_ "embed"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

//go:embed acl.yaml
var aclYaml []byte

type acl struct {
	Roles []struct {
		Name        string   `yaml:"name"`
		Title       string   `yaml:"title"`
		Permissions []string `yaml:"permissions"`
	} `yaml:"roles"`
}

type Manager struct {
	// rolePermissions is a map from role to permissions. Key is "roles/{role}".
	rolePermissions map[string]map[Permission]bool
	groupMembers    map[string]map[string]bool
	PredefinedRoles []*store.RoleMessage
	store           *store.Store
	licenseService  enterprise.LicenseService
}

func NewManager(store *store.Store, licenseService enterprise.LicenseService) (*Manager, error) {
	predefinedRoles, err := loadPredefinedRoles()
	if err != nil {
		return nil, err
	}

	m := &Manager{
		PredefinedRoles: predefinedRoles,
		store:           store,
		licenseService:  licenseService,
	}
	return m, nil
}

// Check if the user has permission on the resource hierarchy.
// When multiple projects are specified, the user should have permission on every projects.
func (m *Manager) CheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	if m.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
		// nolint
		return true, nil
	}

	policyMessage, err := m.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, err
	}
	if ok := check(user.ID, p, policyMessage.Policy, m.rolePermissions, m.groupMembers); ok {
		return true, nil
	}

	if len(projectIDs) > 0 {
		allOK := true
		for _, projectID := range projectIDs {
			project, err := m.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return false, err
			}
			if project == nil {
				return false, errors.Errorf("project %q not found", projectID)
			}
			policyMessage, err := m.store.GetProjectIamPolicy(ctx, project.UID)
			if err != nil {
				return false, err
			}
			if ok := check(user.ID, p, policyMessage.Policy, m.rolePermissions, m.groupMembers); !ok {
				allOK = false
				break
			}
		}
		return allOK, nil
	}
	return false, nil
}

func (m *Manager) ReloadCache(ctx context.Context) error {
	roles, err := m.store.ListRoles(ctx)
	if err != nil {
		return err
	}
	roles = append(roles, m.PredefinedRoles...)

	rolePermissions := make(map[string]map[Permission]bool)
	for _, role := range roles {
		rolePermissions[common.FormatRole(role.ResourceID)] = role.Permissions
	}
	m.rolePermissions = rolePermissions

	groups, err := m.store.ListGroups(ctx, &store.FindGroupMessage{})
	if err != nil {
		return err
	}
	groupMembers := make(map[string]map[string]bool)
	for _, group := range groups {
		usersSet := make(map[string]bool)
		for _, m := range group.Payload.GetMembers() {
			usersSet[m.Member] = true
		}
		groupName := common.FormatGroupEmail(group.Email)
		groupMembers[groupName] = usersSet
	}
	m.groupMembers = groupMembers
	return nil
}

// GetPermissions returns all permissions for the given role.
// Role format is roles/{role}.
func (m *Manager) GetPermissions(role string) (map[Permission]bool, error) {
	permissions, ok := m.rolePermissions[role]
	if !ok {
		return nil, nil
	}
	return permissions, nil
}

func check(userID int, p Permission, policy *storepb.IamPolicy, rolePermissions map[string]map[Permission]bool, groupMembers map[string]map[string]bool) bool {
	userName := common.FormatUserUID(userID)
	for _, binding := range policy.GetBindings() {
		permissions, ok := rolePermissions[binding.GetRole()]
		if !ok {
			continue
		}
		if !permissions[p] {
			continue
		}
		for _, member := range binding.GetMembers() {
			if member == api.AllUsers {
				return true
			}
			if member == userName {
				return true
			}
			if strings.HasPrefix(member, common.GroupPrefix) {
				if groupMembers, ok := groupMembers[member]; ok {
					if groupMembers[userName] {
						return true
					}
				}
			}
		}
	}
	return false
}

func loadPredefinedRoles() ([]*store.RoleMessage, error) {
	predefinedACL := new(acl)
	if err := yaml.Unmarshal(aclYaml, predefinedACL); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal predefined acl")
	}
	var roles []*store.RoleMessage
	for _, role := range predefinedACL.Roles {
		resourceID, err := common.GetRoleID(role.Name)
		if err != nil {
			return nil, err
		}
		permissions := make(map[string]bool)
		for _, p := range role.Permissions {
			permissions[p] = true
		}
		roles = append(roles, &store.RoleMessage{
			CreatorID:   api.SystemBotID,
			ResourceID:  resourceID,
			Name:        role.Title,
			Description: "",
			Permissions: permissions,
		})
	}
	return roles, nil
}
