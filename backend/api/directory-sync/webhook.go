package directorysync

import (
	"context"
	"crypto/subtle"
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
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
	entraIDSource = "Entra ID"
	oktaSource    = "Okta"
)

// detectSCIMSource detects the identity provider source based on User-Agent header.
// Okta sends "Okta SCIM Client" in User-Agent, Azure sends "Azure" or "Microsoft".
func detectSCIMSource(c echo.Context) string {
	userAgent := c.Request().Header.Get("User-Agent")
	userAgentLower := strings.ToLower(userAgent)
	if strings.Contains(userAgentLower, "okta") {
		return oktaSource
	}
	return entraIDSource
}

// scimAuthMiddleware validates authentication and license for all SCIM endpoints.
func (s *Service) scimAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusUnauthorized, err.Error())
		}
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DIRECTORY_SYNC); err != nil {
			return c.String(http.StatusForbidden, err.Error())
		}
		return next(c)
	}
}

// https://developer.xurrent.com/v1/scim/service_provider_config/
// https://scim.cloud/
// https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups
// https://developer.okta.com/docs/reference/scim/scim-20/
func (s *Service) RegisterDirectorySyncRoutes(g *echo.Group) {
	// Apply authentication and license check middleware to all SCIM endpoints.
	g.Use(s.scimAuthMiddleware)

	// ServiceProviderConfig endpoint allows SCIM clients to discover server capabilities.
	// This is required by Okta and recommended by the SCIM 2.0 specification.
	// Docs: https://datatracker.ietf.org/doc/html/rfc7644#section-4
	g.GET("/workspaces/:workspaceID/ServiceProviderConfig", func(c echo.Context) error {
		config := &ServiceProviderConfig{
			Schemas: []string{
				"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig",
			},
			DocumentationURI: "https://www.bytebase.com/docs/administration/scim/overview",
			Patch: ServiceProviderConfigPatch{
				Supported: true,
			},
			Bulk: ServiceProviderConfigBulk{
				Supported:      false,
				MaxOperations:  0,
				MaxPayloadSize: 0,
			},
			Filter: ServiceProviderConfigFilter{
				Supported:  true,
				MaxResults: 100,
			},
			ChangePassword: ServiceProviderConfigSupported{
				Supported: false,
			},
			Sort: ServiceProviderConfigSupported{
				Supported: false,
			},
			Etag: ServiceProviderConfigSupported{
				Supported: false,
			},
			AuthenticationSchemes: []AuthenticationScheme{
				{
					Type:        "oauthbearertoken",
					Name:        "OAuth Bearer Token",
					Description: "Authentication scheme using the OAuth Bearer Token Standard",
				},
			},
		}
		return c.JSON(http.StatusOK, config)
	})

	g.POST("/workspaces/:workspaceID/Users", func(c echo.Context) error {
		ctx := c.Request().Context()
		source := detectSCIMSource(c)

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var scimUser SCIMUser
		if err := json.Unmarshal(body, &scimUser); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		// Normalize email to lowercase as Bytebase requires lowercase emails
		normalizedEmail := normalizeEmail(scimUser.UserName)
		user, err := s.store.GetUserByEmail(ctx, normalizedEmail)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get user %s, error %v", scimUser.UserName, err))
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
				Name:          scimUser.DisplayName,
				Email:         normalizedEmail,
				Type:          storepb.PrincipalType_END_USER,
				MemberDeleted: !scimUser.Active,
				PasswordHash:  string(passwordHash),
				Profile: &storepb.UserProfile{
					Source: source,
				},
			})
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to create user "%s", error %v`, scimUser.UserName, err))
			}
			user = newUser
		} else {
			updatedUser, err := s.updateUserFromSCIM(ctx, user, &scimUser, source)
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to update user "%s", error %v`, user.Email, err))
			}
			user = updatedUser
		}

		return c.JSON(http.StatusCreated, convertToSCIMUser(user))
	})

	// Get a single user. The user id is the Bytebase user uid.
	g.GET("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		user, err := s.getUser(ctx, c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, convertToSCIMUser(user))
	})

	// List users. SCIM clients (Okta, Azure) send ?filter=userName eq "{email}" query.
	// userName maps to userPrincipalName in Azure or login in Okta, which is typically the user's email.
	// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#get-user-by-query
	// Docs: https://developer.okta.com/docs/reference/scim/scim-20/
	g.GET("/workspaces/:workspaceID/Users", func(c echo.Context) error {
		ctx := c.Request().Context()
		response := &ListUsersResponse{
			Schemas: []string{
				"urn:ietf:params:scim:api:messages:2.0:ListResponse",
			},
			TotalResults: 0,
			Resources:    []*SCIMUser{},
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
			// Normalize email to lowercase for consistent lookup
			normalizedEmail := normalizeEmail(expr.Value)
			find.Email = &normalizedEmail
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
			response.Resources = append(response.Resources, convertToSCIMUser(user))
		}

		return c.JSON(http.StatusOK, response)
	})

	g.DELETE("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		user, err := s.getUser(ctx, c)
		if err != nil {
			return err
		}

		deleteUser := true
		if _, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
			Delete: &deleteUser,
		}); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete user, error %v", err))
		}

		return c.String(http.StatusNoContent, "")
	})

	// PUT replaces the entire user resource. Required by SCIM 2.0 (RFC 7644 Section 3.5.1).
	// Body format is same as POST (full SCIMUser object).
	g.PUT("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		user, err := s.getUser(ctx, c)
		if err != nil {
			return err
		}

		source := detectSCIMSource(c)
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var scimUser SCIMUser
		if err := json.Unmarshal(body, &scimUser); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		updatedUser, err := s.updateUserFromSCIM(ctx, user, &scimUser, source)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to update user "%s", error %v`, user.Email, err))
		}

		return c.JSON(http.StatusOK, convertToSCIMUser(updatedUser))
	})

	g.PATCH("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
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
			// SCIM PATCH operations are case-insensitive per RFC 7644.
			// Azure uses PascalCase (Replace), Okta uses lowercase (replace).
			opLower := strings.ToLower(op.OP)
			if opLower != "replace" {
				slog.Warn("empty path only supports replace operation", slog.String("operation", op.OP))
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
				// Normalize email to lowercase as Bytebase requires lowercase emails
				normalizedEmail := normalizeEmail(email)
				updateUser.Email = &normalizedEmail
			case "active":
				active, ok := op.Value.(bool)
				if !ok {
					slog.Warn("unsupport value, expect bool", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				isDelete := !active
				updateUser.Delete = &isDelete
			case "":
				// Empty path with replace operation - Okta sends full resource attributes.
				// Per RFC 7644 Section 3.5.2.3: If path is omitted, the value must be a complex
				// attribute containing sub-attributes to replace.
				valueMap, ok := op.Value.(map[string]any)
				if !ok {
					slog.Warn("unsupport value for empty path, expect object", slog.String("operation", op.OP), slog.Any("value", op.Value))
					continue
				}
				// Extract user attributes from the value object
				if displayName, ok := valueMap["displayName"].(string); ok {
					updateUser.Name = &displayName
				}
				if userName, ok := valueMap["userName"].(string); ok {
					normalizedEmail := normalizeEmail(userName)
					updateUser.Email = &normalizedEmail
				}
				if active, ok := valueMap["active"].(bool); ok {
					isDelete := !active
					updateUser.Delete = &isDelete
				}
				// Handle nested name object (name.givenName, name.familyName)
				if nameObj, ok := valueMap["name"].(map[string]any); ok {
					var nameParts []string
					if givenName, ok := nameObj["givenName"].(string); ok {
						nameParts = append(nameParts, givenName)
					}
					if familyName, ok := nameObj["familyName"].(string); ok {
						nameParts = append(nameParts, familyName)
					}
					if len(nameParts) > 0 {
						fullName := strings.Join(nameParts, " ")
						updateUser.Name = &fullName
					}
				}
			default:
				slog.Warn("unsupport patch path", slog.String("operation", op.OP), slog.String("path", op.Path))
			}
		}

		updatedUser, err := s.store.UpdateUser(ctx, user, updateUser)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update user, error %v", err))
		}

		return c.JSON(http.StatusOK, convertToSCIMUser(updatedUser))
	})

	g.POST("/workspaces/:workspaceID/Groups", func(c echo.Context) error {
		ctx := c.Request().Context()
		source := detectSCIMSource(c)

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var scimGroup SCIMGroup
		if err := json.Unmarshal(body, &scimGroup); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		// SCIM sync group process:
		// - Azure: POST group without members, then PATCH to add members
		// - Okta: May POST group with members directly
		var members []*storepb.GroupMember
		for _, scimMember := range scimGroup.Members {
			member, err := s.getGroupMember(ctx, scimMember)
			if err != nil {
				slog.Error("failed to get scim group member", slog.Any("member", scimMember), log.BBError(err))
				continue
			}
			members = append(members, member)
		}

		group, err := s.store.CreateGroup(ctx, &store.GroupMessage{
			ID:    scimGroup.ExternalID,
			Email: scimGroup.Email,
			Title: scimGroup.DisplayName,
			Payload: &storepb.GroupPayload{
				Source:  source,
				Members: members,
			},
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to create group, error %v", err))
		}

		return c.JSON(http.StatusCreated, convertToSCIMGroup(group))
	})

	// Get a single group. The group id is the Bytebase group resource id.
	g.GET("/workspaces/:workspaceID/Groups/:groupID", func(c echo.Context) error {
		ctx := c.Request().Context()
		group, err := s.getGroup(ctx, c)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, convertToSCIMGroup(group))
	})

	// List groups. SCIM clients (Okta, Azure) send ?filter=externalId eq "{value}" query.
	// externalId can be the IdP's group ID or group email depending on customer's attribute mapping:
	//   - Default: objectId/groupId -> externalId (recommended, stable across email changes)
	//   - Legacy mapping: mail -> externalId (for backward compatibility)
	// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#get-group-by-query
	// Docs: https://developer.okta.com/docs/reference/scim/scim-20/
	g.GET("/workspaces/:workspaceID/Groups", func(c echo.Context) error {
		ctx := c.Request().Context()
		response := &ListGroupsResponse{
			Schemas: []string{
				"urn:ietf:params:scim:api:messages:2.0:ListResponse",
			},
			TotalResults: 0,
			Resources:    []*SCIMGroup{},
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
			// externalId can be either the IdP's group ID or group email, depending on customer's attribute mapping.
			// - Default: objectId/groupId -> externalId (UUID format, no @)
			// - Legacy mapping: mail -> externalId (email format, contains @)
			if strings.Contains(expr.Value, "@") {
				find.Email = &expr.Value
			} else {
				find.ID = &expr.Value
			}
		}

		groups, err := s.store.ListGroups(ctx, find)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to list group, error %v`, err))
		}

		for _, group := range groups {
			response.TotalResults++
			response.Resources = append(response.Resources, convertToSCIMGroup(group))
		}

		return c.JSON(http.StatusOK, response)
	})

	g.DELETE("/workspaces/:workspaceID/Groups/:groupID", func(c echo.Context) error {
		ctx := c.Request().Context()
		group, err := s.getGroup(ctx, c)
		if err != nil {
			return err
		}

		if err := s.store.DeleteGroup(ctx, group.ID); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete group, error %v", err))
		}

		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to reload iam cache, error %v", err))
		}

		return c.JSON(http.StatusNoContent, "")
	})

	// PUT replaces the entire group resource. Required by SCIM 2.0 (RFC 7644 Section 3.5.1).
	// Body format is same as POST (full SCIMGroup object).
	g.PUT("/workspaces/:workspaceID/Groups/:groupID", func(c echo.Context) error {
		ctx := c.Request().Context()
		group, err := s.getGroup(ctx, c)
		if err != nil {
			return err
		}

		source := detectSCIMSource(c)
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var scimGroup SCIMGroup
		if err := json.Unmarshal(body, &scimGroup); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		updatedGroup, err := s.updateGroupFromSCIM(ctx, group, &scimGroup, source)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update group, error %v", err))
		}

		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to reload iam cache, error %v", err))
		}

		return c.JSON(http.StatusOK, convertToSCIMGroup(updatedGroup))
	})

	g.PATCH("/workspaces/:workspaceID/Groups/:groupID", func(c echo.Context) error {
		ctx := c.Request().Context()
		group, err := s.getGroup(ctx, c)
		if err != nil {
			return err
		}

		source := detectSCIMSource(c)
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var patch PatchRequest
		if err := json.Unmarshal(body, &patch); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal body, error %v", err))
		}

		updateGroup := &store.UpdateGroupMessage{
			ID: group.ID,
		}
		for _, op := range patch.Operations {
			// SCIM PATCH operations are case-insensitive per RFC 7644.
			opLower := strings.ToLower(op.OP)

			switch op.Path {
			case "members":
				values, ok := op.Value.([]any)
				if !ok {
					slog.Warn("unsupport value, expect SCIMMember slice", slog.Any("value", op.Value), slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}

				updateGroup.Payload = group.Payload
				updateGroup.Payload.Source = source

				// For replace operation, clear existing members first
				if opLower == "replace" {
					updateGroup.Payload.Members = []*storepb.GroupMember{}
				}

				for _, value := range values {
					member, err := s.getGroupMember(ctx, value)
					if err != nil {
						slog.Error("failed to get scim group member", slog.Any("value", value), log.BBError(err))
						continue
					}

					// Azure uses PascalCase (Add/Remove), Okta uses lowercase (add/remove).
					switch opLower {
					case "add":
						// For add: only add if not already a member
						index := slices.IndexFunc(updateGroup.Payload.Members, func(m *storepb.GroupMember) bool {
							return m.Member == member.Member
						})
						if index < 0 {
							updateGroup.Payload.Members = append(updateGroup.Payload.Members, member)
						}
					case "remove":
						index := slices.IndexFunc(updateGroup.Payload.Members, func(m *storepb.GroupMember) bool {
							return m.Member == member.Member
						})
						if index >= 0 {
							updateGroup.Payload.Members = slices.Delete(updateGroup.Payload.Members, index, index+1)
						}
					case "replace":
						// For replace: members already cleared, just add all
						updateGroup.Payload.Members = append(updateGroup.Payload.Members, member)
					default:
						slog.Warn("unsupport operation type for members", slog.String("operation", op.OP), slog.String("path", op.Path))
						continue
					}
				}
			case "displayName":
				if opLower != "replace" {
					slog.Warn("unsupport operation type for displayName", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				displayName, ok := op.Value.(string)
				if !ok {
					slog.Warn("unsupport value, expect string", slog.String("operation", op.OP), slog.String("path", op.Path))
					continue
				}
				updateGroup.Title = &displayName
			case "":
				// Empty path with replace operation - Okta sends full resource attributes.
				// Per RFC 7644 Section 3.5.2.3: If path is omitted, the value must be a complex
				// attribute containing sub-attributes to replace.
				if opLower != "replace" {
					slog.Warn("empty path only supports replace operation", slog.String("operation", op.OP))
					continue
				}
				valueMap, ok := op.Value.(map[string]any)
				if !ok {
					slog.Warn("unsupport value for empty path, expect object", slog.String("operation", op.OP), slog.Any("value", op.Value))
					continue
				}
				// Extract group attributes from the value object
				if displayName, ok := valueMap["displayName"].(string); ok {
					updateGroup.Title = &displayName
				}
				// Handle members array in the value object
				if members, ok := valueMap["members"].([]any); ok {
					updateGroup.Payload = group.Payload
					updateGroup.Payload.Source = source
					// Clear existing members for full replacement
					updateGroup.Payload.Members = []*storepb.GroupMember{}

					for _, memberValue := range members {
						member, err := s.getGroupMember(ctx, memberValue)
						if err != nil {
							slog.Error("failed to get scim group member", slog.Any("value", memberValue), log.BBError(err))
							continue
						}
						updateGroup.Payload.Members = append(updateGroup.Payload.Members, member)
					}
				}
			default:
				slog.Warn("unsupport patch path", slog.String("operation", op.OP), slog.String("path", op.Path))
			}
		}

		updatedGroup, err := s.store.UpdateGroup(ctx, updateGroup)
		if err != nil {
			slog.Error("failed to update group", log.BBError(err), slog.String("group", updateGroup.ID))
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update group, error %v", err))
		}
		// Reload IAM cache to make sure the group members are updated.
		if err := s.iamManager.ReloadCache(ctx); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to reload iam cache, error %v", err))
		}

		return c.JSON(http.StatusOK, convertToSCIMGroup(updatedGroup))
	})
}

func (s *Service) validRequestURL(ctx context.Context, c echo.Context) error {
	authorization := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
	if authorization == "" {
		return errors.Errorf("missing authorization token")
	}

	workspaceID := c.Param("workspaceID")
	systemSetting, err := s.store.GetSystemSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get system setting")
	}
	myWorkspaceID := systemSetting.WorkspaceId
	if myWorkspaceID != workspaceID {
		return errors.Errorf("invalid workspace id %q, my ID %q", workspaceID, myWorkspaceID)
	}

	workspaceProfileSetting, err := s.store.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get workspace profile setting")
	}
	if workspaceProfileSetting.DirectorySyncToken == "" {
		return errors.Errorf("directory sync token is not configured")
	}

	if subtle.ConstantTimeCompare([]byte(workspaceProfileSetting.DirectorySyncToken), []byte(authorization)) != 1 {
		return errors.Errorf("invalid authorization token")
	}

	return nil
}

func (s *Service) getUser(ctx context.Context, c echo.Context) (*store.UserMessage, error) {
	uid, err := strconv.Atoi(c.Param("userID"))
	if err != nil {
		return nil, c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse user id, error %v", err))
	}

	user, err := s.store.GetUserByID(ctx, uid)
	if err != nil {
		return nil, c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get user, error %v", err))
	}
	if user == nil {
		// PUT must not create new resources per RFC 7644
		return nil, c.JSON(http.StatusNotFound, map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
			"status":  "404",
		})
	}
	return user, nil
}

func (s *Service) getGroup(ctx context.Context, c echo.Context) (*store.GroupMessage, error) {
	groupName, err := decodeGroupIdentifier(c.Param("groupID"))
	if err != nil {
		return nil, c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse group %v, error %v", c.Param("groupID"), err))
	}
	group, err := utils.GetGroupByName(ctx, s.store, common.FormatGroupEmail(groupName))
	if err != nil {
		return nil, c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find group, error %v", err))
	}
	if group == nil {
		return nil, c.JSON(http.StatusNotFound, map[string]any{
			"schemas": []string{
				"urn:ietf:params:scim:api:messages:2.0:Error",
			},
			"status": "404",
		})
	}
	return group, nil
}

func decodeGroupIdentifier(groupID string) (string, error) {
	identifier, err := url.QueryUnescape(groupID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to decode group id %s", groupID)
	}
	return identifier, nil
}

func (s *Service) getGroupMember(ctx context.Context, memberValue any) (*storepb.GroupMember, error) {
	var scimMember SCIMMember
	bytes, err := json.Marshal(memberValue)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &scimMember); err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(scimMember.Value)
	if err != nil {
		return nil, err
	}
	user, err := s.store.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.Errorf("cannot find user %v", uid)
	}

	return &storepb.GroupMember{
		Member: common.FormatUserEmail(user.Email),
		Role:   storepb.GroupMember_MEMBER,
	}, nil
}

// updateUserFromSCIM updates an existing user with data from SCIMUser.
// Used by both POST (when user exists) and PUT operations.
func (s *Service) updateUserFromSCIM(ctx context.Context, user *store.UserMessage, scimUser *SCIMUser, source string) (*store.UserMessage, error) {
	normalizedEmail := normalizeEmail(scimUser.UserName)
	deleted := !scimUser.Active
	patch := &store.UpdateUserMessage{
		Delete:  &deleted,
		Name:    &scimUser.DisplayName,
		Email:   &normalizedEmail,
		Profile: user.Profile,
	}
	patch.Profile.Source = source
	return s.store.UpdateUser(ctx, user, patch)
}

// updateGroupFromSCIM updates an existing group with data from SCIMGroup.
// Used by both POST (when group exists) and PUT operations.
func (s *Service) updateGroupFromSCIM(ctx context.Context, group *store.GroupMessage, scimGroup *SCIMGroup, source string) (*store.GroupMessage, error) {
	var members []*storepb.GroupMember
	for _, scimMember := range scimGroup.Members {
		member, err := s.getGroupMember(ctx, scimMember)
		if err != nil {
			slog.Error("failed to get scim group member", slog.Any("member", scimMember), log.BBError(err))
			continue
		}
		members = append(members, member)
	}

	return s.store.UpdateGroup(ctx, &store.UpdateGroupMessage{
		ID:    group.ID,
		Title: &scimGroup.DisplayName,
		Payload: &storepb.GroupPayload{
			Source:  source,
			Members: members,
		},
	})
}

func convertToSCIMUser(user *store.UserMessage) *SCIMUser {
	return &SCIMUser{
		Schemas: []string{
			"urn:ietf:params:scim:schemas:core:2.0:User",
		},
		UserName:    user.Email,
		Active:      !user.MemberDeleted,
		DisplayName: user.Name,
		ID:          fmt.Sprintf("%d", user.ID),
		Emails: []*SCIMUserEmail{
			{
				Type:    "work",
				Primary: true,
				Value:   user.Email,
			},
		},
		Meta: &SCIMResourceMeta{
			ResourceType: "User",
		},
	}
}

func convertToSCIMGroup(group *store.GroupMessage) *SCIMGroup {
	// Return empty members array. IdPs track members on their side and use PATCH to update.
	// Note: Returning actual members would require looking up user IDs for each member email,
	// which is expensive and not typically needed since IdPs don't rely on GET responses for member state.
	return &SCIMGroup{
		Schemas: []string{
			"urn:ietf:params:scim:schemas:core:2.0:Group",
		},
		// We use the IdP's group object id (external id) to create the group.
		// So both ID and ExternalID should be the group.ID (equals external id).
		ID:          group.ID,
		ExternalID:  group.ID,
		Email:       group.Email,
		DisplayName: group.Title,
		Members:     []*SCIMMember{},
		Meta: &SCIMResourceMeta{
			ResourceType: "Group",
		},
	}
}

// normalizeEmail converts email to lowercase to ensure consistency.
// Bytebase requires all emails to be lowercase for proper user lookup and authentication.
func normalizeEmail(email string) string {
	return strings.ToLower(email)
}
