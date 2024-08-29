package directorysync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"

	"github.com/labstack/echo/v4"

	v1api "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

// https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups
func (s *Service) RegisterDirectorySyncRoutes(g *echo.Group) {
	g.POST("/workspaces/:workspaceID/Users", func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find project, error %v", err))
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
				Type:          api.EndUser,
				MemberDeleted: !aadUser.Active,
				PasswordHash:  string(passwordHash),
			}, api.SystemBotID)
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to create user "%s", error %v`, aadUser.UserName, err))
			}
			user = newUser
		} else if user.MemberDeleted && aadUser.Active {
			deleted := !aadUser.Active
			updatedUser, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{
				Delete: &deleted,
			}, api.SystemBotID)
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf(`failed to update user "%s", error %v`, user.Email, err))
			}
			user = updatedUser
		}

		return c.JSON(http.StatusCreated, formatAADUser(user))
	})

	g.GET("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find project, error %v", err))
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

		return c.JSON(http.StatusOK, formatAADUser(user))
	})

	g.GET("/workspaces/:workspaceID/Users", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find project, error %v", err))
		}

		// AAD SCIM will send ?filter=userName eq "{user name}" query
		filters, err := v1api.ParseFilter(c.QueryParam("filter"))
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse filter, error %v", err))
		}

		response := &ListUsersResponse{
			Schemas: []string{
				"urn:ietf:params:scim:api:messages:2.0:ListResponse",
			},
			TotalResults: 0,
			Resources:    []*AADUser{},
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
			response.Resources = append(response.Resources, formatAADUser(user))
		}

		return c.JSON(http.StatusOK, response)
	})

	g.DELETE("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find project, error %v", err))
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
		}, api.SystemBotID); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update user, error %v", err))
		}

		return c.String(http.StatusNoContent, "")
	})

	g.PATCH("/workspaces/:workspaceID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		if err := s.validRequestURL(ctx, c); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find project, error %v", err))
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		var patch PatchUserRequest
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

		updatedUser, err := s.store.UpdateUser(ctx, user, updateUser, api.SystemBotID)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update user, error %v", err))
		}

		return c.JSON(http.StatusOK, formatAADUser(updatedUser))
	})
}

func formatAADUser(user *store.UserMessage) *AADUser {
	return &AADUser{
		Schemas: []string{
			"urn:ietf:params:scim:schemas:core:2.0:User",
		},
		UserName:    user.Email,
		Active:      !user.MemberDeleted,
		DisplayName: user.Name,
		ID:          common.FormatUserEmail(user.Email),
		Emails: []*AADUserEmail{
			{
				Type:    "work",
				Primary: true,
				Value:   user.Email,
			},
		},
	}
}

func (s *Service) validRequestURL(ctx context.Context, c echo.Context) error {
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

	// TODO: validate token
	return nil
}
