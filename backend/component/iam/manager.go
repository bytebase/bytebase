package iam

import (
	"context"
	_ "embed"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"
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
	workspaceRoles := m.getWorkspaceRoles(user)
	projectRoles, err := m.getProjectRoles(ctx, user, projectIDs)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project roles")
	}

	return m.hasPermission(p, workspaceRoles, projectRoles), nil
}

// GetPermissions returns all permissions for the given role.
// Role format is roles/{role}.
func (m *Manager) GetPermissions(role string) []Permission {
	return m.roles[role]
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
	if len(projectRoles) == 0 {
		return false
	}
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
	conditionInput := getConditionInput()
	var roles []string
	for _, binding := range policy.Bindings {
		ok, err := evalMemberCondition(binding.Condition.Expression, conditionInput)
		if err != nil {
			slog.Error("failed to eval member condition", log.BBError(err))
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

func getConditionInput() map[string]any {
	return map[string]any{
		"request.time": time.Now(),
	}
}

func evalMemberCondition(expr string, input map[string]any) (bool, error) {
	e, err := cel.NewEnv(common.ProjectMemberCELAttributes...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to new cel env")
	}
	ast, iss := e.Compile(expr)
	if iss != nil && iss.Err() != nil {
		return false, errors.Wrapf(iss.Err(), "failed to compile expr %q", expr)
	}
	// enable partial evaluation because the input only has request.time
	// but the expression can have more.
	prg, err := e.Program(ast, cel.EvalOptions(cel.OptPartialEval))
	if err != nil {
		return false, errors.Wrapf(iss.Err(), "failed to construct program")
	}
	vars, err := e.PartialVars(input)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get vars")
	}
	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, errors.Wrapf(err, "failed to eval cel expr")
	}
	// `out` is one of
	// - True
	// - False
	// - a residual expression.

	// return true if the result is a residual expression
	// which means that it passes "the request.time < xxx" check.
	if !celtypes.IsBool(out) {
		return true, nil
	}

	res, ok := out.Equal(celtypes.True).Value().(bool)
	if !ok {
		return false, errors.Errorf("failed to convert cel result to bool")
	}
	return res, nil
}

func isNumber(v string) (int, bool) {
	n, err := strconv.Atoi(v)
	if err == nil {
		return int(n), true
	}
	return 0, false
}
