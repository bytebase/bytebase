package directorysync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"

	"github.com/labstack/echo/v4"

	v1api "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const entraIDSource = "Entra ID"

// https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups
func (s *Service) RegisterDirectorySyncRoutes(g *echo.Group) {
	g.POST("/workspaces/:workspaceID/Users", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var aadUser AADUser
		if err := json.Unmarshal(body, &aadUser); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		user, err := s.store.GetUserByEmail(ctx, aadUser.UserName)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get user %s, error %v", aadUser.UserName, err))
		}
		if user == nil {
			password, err := common.RandomString(20)
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to generate random password, error %v", err))
			}
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to generate password hash, error %v", err))
			}
			newUser, err := s.store.CreateUser(ctx, &store.UserMessage{
				Name:          aadUser.DisplayName,
				Email:         aadUser.UserName,
				Type:          base.EndUser,
				MemberDeleted: !aadUser.Active,
				PasswordHash:  string(passwordHash),
				Profile: &storepb.UserProfile{
					Source: entraIDSource,
				},
			})
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to create user "%s", error %v`, aadUser.UserName, err))
			}
			user = newUser
		} else {
			deleted := !aadUser.Active
			updatedUser, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
				Delete: &deleted,
				Name:   &aadUser.UserName,
				Profile: &storepb.UserProfile{
					Source:                 entraIDSource,
					LastLoginTime:          user.Profile.LastLoginTime,
					LastChangePasswordTime: user.Profile.LastChangePasswordTime,
				},
			})
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to update user "%s", error %v`, user.Email, err))
			}
			user = updatedUser
		}

		return c.JSON(http.StatusCreated, convertToAADUser(user))
	})

	// Get a single user. The user id is the Bytebase user uid.
	g.GET("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		uid, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse user id, error %v", err))
		}

		user, err := s.store.GetUserByID(ctx, uid)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get user, error %v", err))
		}
		if user == nil || user.MemberDeleted {
			return c.JSON(http.StatusNotFound, map[string]any{
				"schemas": []string{
					"urn:ietf:params:scim:api:messages:2.0:Error",
				},
				"status": "404",
			})
		}

		return c.JSON(http.StatusOK, convertToAADUser(user))
	})

	// List users. AAD SCIM will send ?filter=userName eq "{user name}" query
	g.GET("/workspaces/:workspaceID/Users", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		response := &ListUsersResponse{
			Schemas: []string{
				"urn:ietf:params:scim:api:messages:2.0:ListResponse",
			},
			TotalResults: 0,
			Resources:    []*AADUser{},
		}

		filter := c.QueryParam("filter")
		if filter == "" {
			return c.JSON(http.StatusOK, response)
		}

		filters, err := v1api.ParseFilter(filter)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse filter, error %v", err))
		}

		find := &store.FindUserMessage{}
		for _, expr := range filters {
			if expr.Operator != v1api.ComparatorTypeEqual {
				slog.Warn("unsupport filter operation", slog.String("key", expr.Key), slog.String("operator", string(expr.Operator)), slog.String("value", expr.Value))
				continue
			}
			if expr.Key != "userName" {
				slog.Warn("unsupport filter key", slog.String("key", expr.Key), slog.String("operator", string(expr.Operator)), slog.String("value", expr.Value))
				continue
			}
			find.Email = &expr.Value
		}

		users, err := s.store.ListUsers(ctx, find)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to list user, error %v`, err))
		}

		for _, user := range users {
			if user.MemberDeleted {
				continue
			}
			response.TotalResults++
			response.Resources = append(response.Resources, convertToAADUser(user))
		}

		return c.JSON(http.StatusOK, response)
	})

	g.DELETE("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		uid, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse user id, error %v", err))
		}

		user, err := s.store.GetUserByID(ctx, uid)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get user, error %v", err))
		}
		if user == nil || user.MemberDeleted {
			return c.String(http.StatusNoContent, "")
		}

		deleteUser := true
		if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
			Delete: &deleteUser,
		}); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete user, error %v", err))
		}

		return c.String(http.StatusNoContent, "")
	})

	g.PATCH("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var patch PatchRequest
		if err := json.Unmarshal(body, &patch); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		uid, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse user id, error %v", err))
		}

		user, err := s.store.GetUserByID(ctx, uid)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get user, error %v", err))
		}
		if user == nil {
			return c.String(http.StatusNotFound, "cannot found user")
		}

		updateUser := &store.UpdateUserMessage{}
		for _, op := range patch.Operations {
			if op.OP != "Replace" {
				slog.Warn("unsupport operation type", slog.String("operation", op.OP), slog.String("path", op.Path))
				continue
			}
			switch op.Path {
			case "displayName":
				displayName, ok := op.Value.(string)
				if !ok {
					slog.Warn("unsupport value, expect string", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				updateUser.Name = &displayName
			case "userName":
				email, ok := op.Value.(string)
				if !ok {
					slog.Warn("unsupport value, expect string", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				updateUser.Email = &email
			case "active":
				active, ok := op.Value.(bool)
				if !ok {
					slog.Warn("unsupport value, expect bool", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				isDelete := !active
				updateUser.Delete = &isDelete
			default:
				slog.Warn("unsupport patch", slog.String("operation", op.OP), slog.String("path", op.Path))
			}
		}

		updatedUser, err := s.store.UpdateUser(ctx, user, updateUser)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update user, error %v", err))
		}

		return c.JSON(http.StatusOK, convertToAADUser(updatedUser))
	})

	g.POST("/workspaces/:workspaceID/Groups", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var aadGroup AADGroup
		if err := json.Unmarshal(body, &aadGroup); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		// Azure SCIM sync group process:
		// 1. POST users
		// 2. POST group without members
		// 3. PATCH group with members
		group, err := s.store.CreateGroup(ctx, &store.GroupMessage{
			Email: aadGroup.ExternalID,
			Title: aadGroup.DisplayName,
			Payload: &storepb.GroupPayload{
				Source: entraIDSource,
			},
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to create group, error %v", err))
		}

		return c.JSON(http.StatusCreated, convertToAADGroup(group))
	})

	// Get a single group. The group id is the Bytebase group resource id.
	g.GET("/workspaces/:workspaceID/Groups/:groupID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		email, err := decodeGroupEmail(c.Param("groupID"))
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		group, err := s.store.GetGroup(ctx, email)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find group, error %v", err))
		}
		if group == nil {
			return c.JSON(http.StatusNotFound, map[string]any{
				"schemas": []string{
					"urn:ietf:params:scim:api:messages:2.0:Error",
				},
				"status": "404",
			})
		}

		return c.JSON(http.StatusOK, convertToAADGroup(group))
	})

	// List groups. AAD SCIM will send ?filter=externalId eq "{group email}" query
	g.GET("/workspaces/:workspaceID/Groups", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		response := &ListGroupsResponse{
			Schemas: []string{
				"urn:ietf:params:scim:api:messages:2.0:ListResponse",
			},
			TotalResults: 0,
			Resources:    []*AADGroup{},
		}

		filter := c.QueryParam("filter")
		if filter == "" {
			return c.JSON(http.StatusOK, response)
		}

		filters, err := v1api.ParseFilter(filter)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse filter, error %v", err))
		}

		find := &store.FindGroupMessage{}
		for _, expr := range filters {
			if expr.Operator != v1api.ComparatorTypeEqual {
				slog.Warn("unsupport filter operation", slog.String("key", expr.Key), slog.String("operator", string(expr.Operator)), slog.String("value", expr.Value))
				continue
			}
			if expr.Key != "externalId" {
				slog.Warn("unsupport filter key", slog.String("key", expr.Key), slog.String("operator", string(expr.Operator)), slog.String("value", expr.Value))
				continue
			}
			find.Email = &expr.Value
		}

		groups, err := s.store.ListGroups(ctx, find)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to list group, error %v`, err))
		}

		for _, group := range groups {
			response.TotalResults++
			response.Resources = append(response.Resources, convertToAADGroup(group))
		}

		return c.JSON(http.StatusOK, response)
	})

	g.DELETE("/workspaces/:workspaceID/Groups/:groupID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		email, err := decodeGroupEmail(c.Param("groupID"))
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		if err := s.store.DeleteGroup(ctx, email); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete group, error %v", err))
		}

		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to reload iam cache, error %v", err))
		}

		return c.JSON(http.StatusNoContent, "")
	})

	g.PATCH("/workspaces/:workspaceID/Groups/:groupID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var patch PatchRequest
		if err := json.Unmarshal(body, &patch); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		email, err := decodeGroupEmail(c.Param("groupID"))
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		group, err := s.store.GetGroup(ctx, email)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find group, error %v", err))
		}
		if group == nil {
			return c.String(http.StatusNotFound, "cannot found group")
		}

		updateGroup := &store.UpdateGroupMessage{}
		for _, op := range patch.Operations {
			switch op.Path {
			case "members":
				values, ok := op.Value.([]any)
				if !ok {
					slog.Warn("unsupport value, expect PatchMember slice", slog.Any("value", op.Value), slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}

				updateGroup.Payload = group.Payload
				updateGroup.Payload.Source = entraIDSource

				for _, value := range values {
					var patchMember PatchMember
					bytes, err := json.Marshal(value)
					if err != nil {
						slog.Warn("failed to marshal patch member", slog.Any("value", value), slog.String("operation", op.OP), slog.String("path", op.Path), log.BBError(err))
						continue
					}
					if err := json.Unmarshal(bytes, &patchMember); err != nil {
						slog.Warn("failed to unmarshal patch member", slog.Any("value", value), slog.String("operation", op.OP), slog.String("path", op.Path), log.BBError(err))
						continue
					}

					// the member identifier in group patch is Bytebase user uid
					uid, err := strconv.Atoi(patchMember.Value)
					if err != nil {
						return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse user id, error %v", err))
					}
					user, err := s.store.GetUserByID(ctx, uid)
					if err != nil {
						return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get user, error %v", err))
					}
					if user == nil {
						slog.Warn("cannot found user", slog.String("operation", op.OP), slog.String("uid", patchMember.Value))
						continue
					}

					member := &storepb.GroupMember{
						Member: common.FormatUserUID(user.ID),
						Role:   storepb.GroupMember_MEMBER,
					}
					index := slices.IndexFunc(group.Payload.Members, func(m *storepb.GroupMember) bool {
						return m.Member == member.Member
					})
					switch op.OP {
					case "Add":
						if index < 0 {
							updateGroup.Payload.Members = append(updateGroup.Payload.Members, member)
						}
					case "Remove":
						if index >= 0 {
							updateGroup.Payload.Members = slices.Delete(updateGroup.Payload.Members, index, index+1)
						}
					default:
						slog.Warn("unsupport operation type", slog.String("operation", op.OP), slog.String("path", op.Path))
						continue
					}
				}
			case "displayName":
				if op.OP != "Replace" {
					slog.Warn("unsupport operation type", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				displayName, ok := op.Value.(string)
				if !ok {
					slog.Warn("unsupport value, expect string", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				updateGroup.Title = &displayName
			default:
				slog.Warn("unsupport patch", slog.String("operation", op.OP), slog.String("path", op.Path))
			}
		}

		updatedGroup, err := s.store.UpdateGroup(ctx, group.Email, updateGroup)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update group, error %v", err))
		}
		// Reload IAM cache to make sure the group members are updated.
		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to reload iam cache, error %v", err))
		}

		return c.JSON(http.StatusOK, convertToAADGroup(updatedGroup))
	})
}

func (s *Service) validRequestURL(ctx context.Context, c echo.Context) error {
	authorization := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
	if authorization == "" {
		return errors.Errorf("missing authorization token")
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return err
	}
	if setting.ExternalUrl == "" {
		return errors.Errorf("external URL is empty")
	}

	workspaceID := c.Param("workspaceID")

	myWorkspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return err
	}
	if myWorkspaceID != workspaceID {
		return errors.Errorf("invalid workspace id %q, my ID %q", workspaceID, myWorkspaceID)
	}

	scimSetting, err := s.store.GetSettingV2(ctx, base.SettingSCIM)
	if err != nil {
		return errors.Wrapf(err, "failed to find scim setting")
	}
	if scimSetting == nil {
		return errors.Errorf("cannot found scim setting")
	}
	payload := new(storepb.SCIMSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(scimSetting.Value), payload); err != nil {
		return errors.Wrapf(err, "failed to unmarshal scim setting")
	}

	if payload.Token != authorization {
		return errors.Errorf("invalid authorization token")
	}

	return nil
}

func decodeGroupEmail(groupID string) (string, error) {
	email, err := url.QueryUnescape(groupID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to decode group id %s", groupID)
	}
	return email, nil
}

func convertToAADUser(user *store.UserMessage) *AADUser {
	return &AADUser{
		Schemas: []string{
			"urn:ietf:params:scim:schemas:core:2.0:User",
		},
		UserName:    user.Email,
		Active:      !user.MemberDeleted,
		DisplayName: user.Name,
		ID:          fmt.Sprintf("%d", user.ID),
		Emails: []*AADUserEmail{
			{
				Type:    "work",
				Primary: true,
				Value:   user.Email,
			},
		},
		Meta: &AADResourceMeta{
			ResourceType: "User",
		},
	}
}

func convertToAADGroup(group *store.GroupMessage) *AADGroup {
	return &AADGroup{
		Schemas: []string{
			"urn:ietf:params:scim:schemas:core:2.0:Group",
		},
		ID:          group.Email,
		ExternalID:  group.Email,
		DisplayName: group.Title,
		Meta: &AADResourceMeta{
			ResourceType: "Group",
		},
	}
}
