package utils

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// GetUsersByRoleInIAMPolicy gets users in the iam policy.
// TODO(p0ny): renovate this function to respect CEL.
func GetUsersByRoleInIAMPolicy(ctx context.Context, stores *store.Store, role api.Role, policy *storepb.ProjectIamPolicy) []*store.UserMessage {
	roleFullName := common.FormatRole(role.String())
	var users []*store.UserMessage

	for _, binding := range policy.Bindings {
		if binding.Role != roleFullName {
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
