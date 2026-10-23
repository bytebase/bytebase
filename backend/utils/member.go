//nolint:revive
package utils

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// ValidateIAMBinding checks if the binding's condition expression is still valid (e.g., not expired).
func ValidateIAMBinding(binding *storepb.Binding) bool {
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

// GetUsersByRoleInIAMPolicy returns the users bound to the role in the
// policies. A failed lookup is logged and its member skipped; a caller whose
// decision changes with a missing user uses UsersByRoleInIAMPolicy.
func GetUsersByRoleInIAMPolicy(
	ctx context.Context,
	stores *store.Store,
	workspace string,
	role string,
	allowAllUsers bool,
	policies ...*storepb.IamPolicy,
) []*store.UserMessage {
	users, err := UsersByRoleInIAMPolicy(ctx, stores, workspace, role, allowAllUsers, policies...)
	if err != nil {
		slog.Error("failed to resolve every user bound to the role", slog.String("role", role), log.BBError(err))
	}
	return users
}

// UsersByRoleInIAMPolicy returns the users bound to the role in the policies,
// and the failed lookups joined into one error. The users that resolved are
// returned either way.
func UsersByRoleInIAMPolicy(
	ctx context.Context,
	stores *store.Store,
	workspace string,
	role string,
	allowAllUsers bool,
	policies ...*storepb.IamPolicy,
) ([]*store.UserMessage, error) {
	roleFullName := role
	if !strings.HasPrefix(role, common.RolePrefix) {
		roleFullName = common.FormatRole(role)
	}
	var users []*store.UserMessage
	var failures []error

	seen := map[string]bool{}
	for _, policy := range policies {
		for _, binding := range policy.Bindings {
			if binding.Role != roleFullName {
				continue
			}

			if !ValidateIAMBinding(binding) {
				continue
			}

			for _, member := range binding.Members {
				if member == common.AllUsers && allowAllUsers {
					// TODO(d): make it more efficient.
					allUsers, err := stores.ListUsers(ctx, &store.FindUserMessage{
						ShowDeleted: false,
					})
					if err != nil {
						failures = append(failures, pkgerrors.Wrapf(err, "failed to list all users for role %s", role))
						continue
					}
					return allUsers, nil
				}
				userMessages, err := usersByMember(ctx, stores, workspace, member)
				if err != nil {
					failures = append(failures, err)
				}

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

	return users, errors.Join(failures...)
}

// GetUsersByMember gets user messages by member.
// The member should be in users/{email}, serviceAccounts/{email}, workloadIdentities/{email}, or groups/{email} format.
// A failed lookup is logged and skipped.
func GetUsersByMember(ctx context.Context, stores *store.Store, workspace string, member string) []*store.UserMessage {
	users, err := usersByMember(ctx, stores, workspace, member)
	if err != nil {
		slog.Error("failed to resolve every user of the member", slog.String("member", member), log.BBError(err))
	}
	return users
}

func usersByMember(ctx context.Context, stores *store.Store, workspace string, member string) ([]*store.UserMessage, error) {
	prefix := ""
	switch {
	case strings.HasPrefix(member, common.UserNamePrefix):
		prefix = common.UserNamePrefix
	case strings.HasPrefix(member, common.ServiceAccountNamePrefix):
		prefix = common.ServiceAccountNamePrefix
	case strings.HasPrefix(member, common.WorkloadIdentityNamePrefix):
		prefix = common.WorkloadIdentityNamePrefix
	case strings.HasPrefix(member, common.GroupPrefix):
		group, err := stores.GetGroupByName(ctx, workspace, member)
		if err != nil {
			return nil, pkgerrors.Wrapf(err, "failed to get group %s", member)
		}
		if group == nil {
			slog.Error("cannot found group", slog.String("group", member))
			return nil, nil
		}
		var users []*store.UserMessage
		var failures []error
		for _, member := range group.Payload.Members {
			user, err := userByIdentifier(ctx, stores, member.Member, common.UserNamePrefix)
			if err != nil {
				failures = append(failures, err)
				continue
			}
			if user != nil {
				users = append(users, user)
			}
		}
		return users, errors.Join(failures...)
	default:
		return nil, nil
	}
	user, err := userByIdentifier(ctx, stores, member, prefix)
	if err != nil || user == nil {
		return nil, err
	}
	return []*store.UserMessage{user}, nil
}

// userByIdentifier gets the user an identifier in {prefix}{email} format names,
// where prefix is users/, serviceAccounts/ or workloadIdentities/. A principal
// that does not exist is nil with no error.
func userByIdentifier(ctx context.Context, stores *store.Store, identifier, prefix string) (*store.UserMessage, error) {
	var email string
	var err error
	switch prefix {
	case common.ServiceAccountNamePrefix:
		email, err = common.GetServiceAccountEmail(identifier)
	case common.WorkloadIdentityNamePrefix:
		email, err = common.GetWorkloadIdentityEmail(identifier)
	default:
		email, err = common.GetUserEmail(identifier)
	}
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to parse email from identifier %s", identifier)
	}
	account, err := stores.GetAccountByEmail(ctx, email)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "failed to get principal %s", identifier)
	}
	if account == nil {
		return nil, nil
	}
	return userMessageFromAccount(account), nil
}

func userMessageFromAccount(account *store.AccountMessage) *store.UserMessage {
	return &store.UserMessage{
		Email:         account.Email,
		Name:          account.Name,
		Type:          account.Type,
		Phone:         account.Phone,
		MemberDeleted: account.MemberDeleted,
	}
}

// GetUserIAMPolicyBindings return the valid bindings for the user.
func GetUserIAMPolicyBindings(ctx context.Context, stores *store.Store, workspace string, user *store.UserMessage, policies ...*storepb.IamPolicy) []*storepb.Binding {
	userMemberName := formatMemberNameByType(user)

	var bindings []*storepb.Binding

	for _, policy := range policies {
		for _, binding := range policy.Bindings {
			if !ValidateIAMBinding(binding) {
				continue
			}

			hasUser := false
			for _, member := range binding.Members {
				if member == common.AllUsers {
					hasUser = true
					break
				}
				if userMemberName == member {
					hasUser = true
					break
				}
				if strings.HasPrefix(member, common.GroupPrefix) {
					group, err := stores.GetGroupByName(ctx, workspace, member)
					if err != nil {
						slog.Error("failed to get group", slog.String("group", member), log.BBError(err))
						continue
					}
					if group == nil {
						slog.Error("cannot found group", slog.String("group", member))
						continue
					}
					for _, member := range group.Payload.Members {
						if userMemberName == member.Member {
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

// MemberContainsUser checks if a member (user, service account, workload identity, or group) contains the specified user.
// The member should be in users/{email}, serviceAccounts/{email}, workloadIdentities/{email}, or groups/{email} format.
func MemberContainsUser(ctx context.Context, stores *store.Store, workspace string, member string, user *store.UserMessage) bool {
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

	// Check if member is a service account
	if strings.HasPrefix(member, common.ServiceAccountNamePrefix) {
		memberEmail, err := common.GetServiceAccountEmail(member)
		if err != nil {
			slog.Error("failed to parse service account email", slog.String("member", member), log.BBError(err))
			return false
		}
		return memberEmail == user.Email
	}

	// Check if member is a workload identity
	if strings.HasPrefix(member, common.WorkloadIdentityNamePrefix) {
		memberEmail, err := common.GetWorkloadIdentityEmail(member)
		if err != nil {
			slog.Error("failed to parse workload identity email", slog.String("member", member), log.BBError(err))
			return false
		}
		return memberEmail == user.Email
	}

	// Check if member is a group
	if strings.HasPrefix(member, common.GroupPrefix) {
		group, err := stores.GetGroupByName(ctx, workspace, member)
		if err != nil {
			slog.Error("failed to get group", slog.String("group", member), log.BBError(err))
			return false
		}
		if group == nil {
			slog.Error("cannot find group", slog.String("group", member))
			return false
		}
		userMemberName := formatMemberNameByType(user)
		for _, groupMember := range group.Payload.Members {
			if userMemberName == groupMember.Member {
				return true
			}
		}
	}

	return false
}

// GetUserRolesInIamPolicy returns the `uniq`ed roles of a user, including workspace roles and the roles in the projects.
// the condition of role binding is respected and evaluated with request.time=time.Now().
// the returned role name should in the roles/{id} format.
func GetUserRolesInIamPolicy(ctx context.Context, stores *store.Store, workspace string, user *store.UserMessage, policies ...*storepb.IamPolicy) []string {
	var roles []string

	for _, policy := range policies {
		bindings := GetUserIAMPolicyBindings(ctx, stores, workspace, user, policy)
		for _, binding := range bindings {
			roles = append(roles, binding.Role)
		}
	}
	roles = common.Uniq(roles)

	return roles
}

// See GetUserRoles. The returned map key format is roles/{role}.
func GetUserFormattedRolesMap(ctx context.Context, stores *store.Store, workspace string, user *store.UserMessage, projectPolicies ...*storepb.IamPolicy) map[string]bool {
	roles := GetUserRolesInIamPolicy(ctx, stores, workspace, user, projectPolicies...)

	rolesMap := make(map[string]bool)
	for _, role := range roles {
		rolesMap[role] = true
	}
	return rolesMap
}
