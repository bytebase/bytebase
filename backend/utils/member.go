//nolint:revive
package utils

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func validateIAMBinding(binding *storepb.Binding) bool {
	ok, err := common.EvalBindingCondition(binding.Condition.GetExpression(), time.Now())
	if err != nil {
		slog.Error("failed to eval binding condition", slog.String("expression", binding.Condition.GetExpression()), log.BBError(err))
		return false
	}
	return ok
}

func FormatGroupName(group *store.GroupMessage) string {
	if group.Email != "" {
		return common.FormatGroupEmail(group.Email)
	}
	return common.FormatGroupEmail(group.ID)
}

// GetUsersByRoleInIAMPolicy gets users in the iam policy.
func GetUsersByRoleInIAMPolicy(ctx context.Context, stores *store.Store, role string, policies ...*storepb.IamPolicy) []*store.UserMessage {
	roleFullName := common.FormatRole(role)
	var users []*store.UserMessage

	seen := map[string]bool{}
	for _, policy := range policies {
		for _, binding := range policy.Bindings {
			if binding.Role != roleFullName {
				continue
			}

			if !validateIAMBinding(binding) {
				continue
			}

			for _, member := range binding.Members {
				if member == common.AllUsers {
					// TODO(d): make it more efficient.
					allUsers, err := stores.ListUsers(ctx, &store.FindUserMessage{
						ShowDeleted: false,
					})
					if err != nil {
						slog.Error("failed to list all users for role", slog.String("role", role), log.BBError(err))
						continue
					}
					return allUsers
				}
				userMessages := GetUsersByMember(ctx, stores, member)

				for _, user := range userMessages {
					if seen[user.Email] {
						continue
					}
					seen[user.Email] = true
					users = append(users, user)
				}
			}
		}
	}

	return users
}

// GetGroupByName finds a group by identifier which can be either:
//   - Azure objectId (UUID format, no @) - used as group ID in new deployments
//   - Group email (contains @) - used as group ID in legacy deployments
//
// This supports both attribute mapping configurations in Azure Entra ID.
func GetGroupByName(ctx context.Context, stores *store.Store, name string) (*store.GroupMessage, error) {
	identifier, err := common.GetGroupEmail(name)
	if err != nil {
		return nil, err
	}
	find := &store.FindGroupMessage{}
	if strings.Contains(identifier, "@") {
		find.Email = &identifier
	} else {
		find.ID = &identifier
	}

	group, err := stores.GetGroup(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get group %q", name)
	}
	return group, nil
}

// GetUsersByMember gets user messages by member.
// The member should be in users/{email} or groups/{email} format.
func GetUsersByMember(ctx context.Context, stores *store.Store, member string) []*store.UserMessage {
	var users []*store.UserMessage
	if strings.HasPrefix(member, common.UserNamePrefix) {
		user := getUserByIdentifier(ctx, stores, member)
		if user != nil {
			users = append(users, user)
		}
	} else if strings.HasPrefix(member, common.GroupPrefix) {
		group, err := GetGroupByName(ctx, stores, member)
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
// The identifier should be in users/{email} format.
func getUserByIdentifier(ctx context.Context, stores *store.Store, identifier string) *store.UserMessage {
	email, err := common.GetUserEmail(identifier)
	if err != nil {
		slog.Error("failed to parse user email", slog.String("user", identifier), log.BBError(err))
		return nil
	}
	user, err := stores.GetUserByEmail(ctx, email)
	if err != nil {
		slog.Error("failed to get user by email", slog.String("user", identifier), log.BBError(err))
		return nil
	}
	return user
}

// GetUserIAMPolicyBindings return the valid bindings for the user.
func GetUserIAMPolicyBindings(ctx context.Context, stores *store.Store, user *store.UserMessage, policies ...*storepb.IamPolicy) []*storepb.Binding {
	userEmailFullName := common.FormatUserEmail(user.Email)

	var bindings []*storepb.Binding

	for _, policy := range policies {
		for _, binding := range policy.Bindings {
			if !validateIAMBinding(binding) {
				continue
			}

			hasUser := false
			for _, member := range binding.Members {
				if member == common.AllUsers {
					hasUser = true
					break
				}
				if userEmailFullName == member {
					hasUser = true
					break
				}
				if strings.HasPrefix(member, common.GroupPrefix) {
					group, err := GetGroupByName(ctx, stores, member)
					if err != nil {
						slog.Error("failed to get group", slog.String("group", member), log.BBError(err))
						continue
					}
					if group == nil {
						slog.Error("cannot found group", slog.String("group", member))
						continue
					}
					for _, member := range group.Payload.Members {
						if userEmailFullName == member.Member {
							hasUser = true
							break
						}
					}
					if hasUser {
						break
					}
				}
			}
			if hasUser {
				bindings = append(bindings, binding)
			}
		}
	}
	return bindings
}

// MemberContainsUser checks if a member (user or group) contains the specified user.
// The member should be in users/{email} or groups/{email} format.
func MemberContainsUser(ctx context.Context, stores *store.Store, member string, user *store.UserMessage) bool {
	if member == common.AllUsers {
		return true
	}

	// Check if member is a user
	if strings.HasPrefix(member, common.UserNamePrefix) {
		memberEmail, err := common.GetUserEmail(member)
		if err != nil {
			slog.Error("failed to parse user email", slog.String("member", member), log.BBError(err))
			return false
		}
		return memberEmail == user.Email
	}

	// Check if member is a group
	if strings.HasPrefix(member, common.GroupPrefix) {
		group, err := GetGroupByName(ctx, stores, member)
		if err != nil {
			slog.Error("failed to get group", slog.String("group", member), log.BBError(err))
			return false
		}
		if group == nil {
			slog.Error("cannot find group", slog.String("group", member))
			return false
		}
		userEmailFullName := common.FormatUserEmail(user.Email)
		for _, groupMember := range group.Payload.Members {
			if userEmailFullName == groupMember.Member {
				return true
			}
		}
	}

	return false
}

// GetUserRolesInIamPolicy returns the `uniq`ed roles of a user, including workspace roles and the roles in the projects.
// the condition of role binding is respected and evaluated with request.time=time.Now().
// the returned role name should in the roles/{id} format.
func GetUserRolesInIamPolicy(ctx context.Context, stores *store.Store, user *store.UserMessage, policies ...*storepb.IamPolicy) []string {
	var roles []string

	for _, policy := range policies {
		bindings := GetUserIAMPolicyBindings(ctx, stores, user, policy)
		for _, binding := range bindings {
			roles = append(roles, binding.Role)
		}
	}
	roles = common.Uniq(roles)

	return roles
}

// See GetUserRoles. The returned map key format is roles/{role}.
func GetUserFormattedRolesMap(ctx context.Context, stores *store.Store, user *store.UserMessage, projectPolicies ...*storepb.IamPolicy) map[string]bool {
	roles := GetUserRolesInIamPolicy(ctx, stores, user, projectPolicies...)

	rolesMap := make(map[string]bool)
	for _, role := range roles {
		rolesMap[role] = true
	}
	return rolesMap
}
