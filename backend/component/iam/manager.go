package iam

import (
	"context"
	_ "embed"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
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
	// member - groups mapping
	memberGroups    map[string][]string
	PredefinedRoles []*store.RoleMessage
	store           *store.Store
	licenseService  *enterprise.LicenseService
}

func NewManager(store *store.Store, licenseService *enterprise.LicenseService) (*Manager, error) {
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
// CEL on the binding is not considered.
// When multiple projects are specified, the user should have permission on every projects.
func (m *Manager) CheckPermission(ctx context.Context, p Permission, user *store.UserMessage, projectIDs ...string) (bool, error) {
	policyMessage, err := m.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return false, err
	}
	if ok := check(user, p, policyMessage.Policy, m.rolePermissions, m.groupMembers); ok {
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
			if ok := check(user, p, policyMessage.Policy, m.rolePermissions, m.groupMembers); !ok {
				allOK = false
				break
			}
		}
		return allOK, nil
	}
	return false, nil
}

func (m *Manager) ReloadCache(ctx context.Context) error {
	roles, err := m.store.ListRoles(ctx, &store.FindRoleMessage{})
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
	memberGroups := make(map[string][]string)
	for _, group := range groups {
		usersSet := make(map[string]bool)
		groupName := utils.FormatGroupName(group)
		for _, m := range group.Payload.GetMembers() {
			usersSet[m.Member] = true
			if _, ok := memberGroups[m.Member]; !ok {
				memberGroups[m.Member] = []string{}
			}
			memberGroups[m.Member] = append(memberGroups[m.Member], groupName)
		}
		groupMembers[groupName] = usersSet
	}
	m.groupMembers = groupMembers
	m.memberGroups = memberGroups
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

func (m *Manager) GetUserGroups(email string) []string {
	return m.memberGroups[common.FormatUserEmail(email)]
}

func check(user *store.UserMessage, p Permission, policy *storepb.IamPolicy, rolePermissions map[string]map[Permission]bool, groupMembers map[string]map[string]bool) bool {
	userName := common.FormatUserEmail(user.Email)

	for _, binding := range policy.GetBindings() {
		permissions, ok := rolePermissions[binding.GetRole()]
		if !ok {
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
			ResourceID:  resourceID,
			Name:        role.Title,
			Description: "",
			Permissions: permissions,
		})
	}
	return roles, nil
}
