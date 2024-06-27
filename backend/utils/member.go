package utils

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func validateIAMBinding(binding *storepb.Binding) bool {
	ok, err := common.EvalBindingCondition(binding.Condition.GetExpression(), time.Now())
	if err != nil {
		slog.Error("failed to eval binding condition", slog.String("expression", binding.Condition.GetExpression()), log.BBError(err))
		return false
	}
	return ok
}

// GetUsersByRoleInIAMPolicy gets users in the iam policy.
func GetUsersByRoleInIAMPolicy(ctx context.Context, stores *store.Store, role api.Role, policy *storepb.ProjectIamPolicy) []*store.UserMessage {
	roleFullName := common.FormatRole(role.String())
	var users []*store.UserMessage

	for _, binding := range policy.Bindings {
		if binding.Role != roleFullName {
			continue
		}

		if !validateIAMBinding(binding) {
			continue
		}

		for _, member := range binding.Members {
			if member == api.AllUsers {
				// TODO(d): make it more efficient.
				allUsers, err := stores.ListUsers(ctx, &store.FindUserMessage{
					ShowDeleted: false,
				})
				if err != nil {
					slog.Error("failed to list all users for role", slog.String("role", role.String()), log.BBError(err))
					continue
				}
				return allUsers
			}
			userMessages := GetUsersByMember(ctx, stores, member)
			users = append(users, userMessages...)
		}
	}

	return users
}

// GetUsersByMember gets user messages by member.
// The member should in users/{uid} or groups/{email} format.
func GetUsersByMember(ctx context.Context, stores *store.Store, member string) []*store.UserMessage {
	var users []*store.UserMessage
	if strings.HasPrefix(member, common.UserNamePrefix) {
		user := getUserByIdentifier(ctx, stores, member)
		if user != nil {
			users = append(users, user)
		}
	} else if strings.HasPrefix(member, common.UserGroupPrefix) {
		groupEmail, err := common.GetUserGroupEmail(member)
		if err != nil {
			slog.Error("failed to parse group email", slog.String("group", member), log.BBError(err))
			return users
		}
		group, err := stores.GetUserGroup(ctx, groupEmail)
		if err != nil {
			slog.Error("failed to get group", slog.String("group", member), log.BBError(err))
			return users
		}
		if group == nil {
			slog.Error("cannot found group", slog.String("group", member))
			return users
		}
		for _, member := range group.Payload.Members {
			user := getUserByIdentifier(ctx, stores, member.Member)
			if user != nil {
				users = append(users, user)
			}
		}
	}
	return users
}

// getUserByIdentifier gets user message by identifier.
// The identifier should in users/{uid} format.
func getUserByIdentifier(ctx context.Context, stores *store.Store, identifier string) *store.UserMessage {
	userUID, err := common.GetUserID(identifier)
	if err != nil {
		slog.Error("failed to parse user id", slog.String("user", identifier), log.BBError(err))
		return nil
	}
	user, err := stores.GetUserByID(ctx, userUID)
	if err != nil {
		slog.Error("failed to get user", slog.String("user", identifier), log.BBError(err))
		return nil
	}
	return user
}

// GetUserIAMPolicyBindings return the valid bindings for the user.
func GetUserIAMPolicyBindings(ctx context.Context, stores *store.Store, user *store.UserMessage, policy *storepb.ProjectIamPolicy) []*storepb.Binding {
	userIDFullName := common.FormatUserUID(user.ID)

	var bindings []*storepb.Binding
	for _, binding := range policy.Bindings {
		if !validateIAMBinding(binding) {
			continue
		}

		hasUser := false
		for _, member := range binding.Members {
			if member == api.AllUsers {
				hasUser = true
				break
			}
			if userIDFullName == member {
				hasUser = true
				break
			}
			if strings.HasPrefix(member, common.UserGroupPrefix) {
				groupEmail, err := common.GetUserGroupEmail(member)
				if err != nil {
					slog.Error("failed to parse group email", slog.String("group", member), log.BBError(err))
					continue
				}
				group, err := stores.GetUserGroup(ctx, groupEmail)
				if err != nil {
					slog.Error("failed to get group", slog.String("group", member), log.BBError(err))
					continue
				}
				if group == nil {
					slog.Error("cannot found group", slog.String("group", member))
					continue
				}
				for _, member := range group.Payload.Members {
					if userIDFullName == member.Member {
						hasUser = true
						break
					}
				}
			}
		}
		if hasUser {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

// getUserRoles returns the `uniq`ed roles of a user, including workspace roles and the roles in the projects.
// the condition of role binding is respected and evaluated with request.time=time.Now().
// the returned role name should in the roles/{id} format.
func getUserRoles(ctx context.Context, stores *store.Store, user *store.UserMessage, projectPolicies ...*storepb.ProjectIamPolicy) ([]string, error) {
	var roles []string
	for _, userRole := range user.Roles {
		roles = append(roles, common.FormatRole(userRole.String()))
	}

	for _, projectPolicy := range projectPolicies {
		bindings := GetUserIAMPolicyBindings(ctx, stores, user, projectPolicy)
		for _, binding := range bindings {
			roles = append(roles, binding.Role)
		}
	}
	roles = uniq(roles)

	return roles, nil
}

// See GetUserRoles.
func GetUserRolesMap(ctx context.Context, stores *store.Store, user *store.UserMessage, projectPolicies ...*storepb.ProjectIamPolicy) (map[api.Role]bool, error) {
	roles, err := getUserRoles(ctx, stores, user, projectPolicies...)
	if err != nil {
		return nil, err
	}

	rolesMap := make(map[api.Role]bool)
	for _, role := range roles {
		rolesMap[api.Role(strings.TrimPrefix(role, "roles/"))] = true
	}
	return rolesMap, nil
}

// See GetUserRoles. The returned map key format is roles/{role}.
func GetUserFormattedRolesMap(ctx context.Context, stores *store.Store, user *store.UserMessage, projectPolicies ...*storepb.ProjectIamPolicy) (map[string]bool, error) {
	roles, err := getUserRoles(ctx, stores, user, projectPolicies...)
	if err != nil {
		return nil, err
	}

	rolesMap := make(map[string]bool)
	for _, role := range roles {
		rolesMap[role] = true
	}
	return rolesMap, nil
}

// BackfillRoleFromRoles finds the highest workspace level role from roles.
func BackfillRoleFromRoles(roles []api.Role) api.Role {
	admin, dba := false, false
	for _, r := range roles {
		if r == api.WorkspaceAdmin {
			admin = true
			break
		}
		if r == api.WorkspaceDBA {
			dba = true
		}
	}
	if admin {
		return api.WorkspaceAdmin
	}
	if dba {
		return api.WorkspaceDBA
	}
	return api.WorkspaceMember
}
