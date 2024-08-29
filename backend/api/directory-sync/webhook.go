package directorysync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/labstack/echo/v4"

	v1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups
func (s *Service) RegisterDirectorySyncRoutes(g *echo.Group) {
	g.POST("/workspaces/:workspaceID/projects/:projectID/Users", func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read body, error %v", err))
		}

		ctx := c.Request().Context()

		project, err := s.validRequestURL(ctx, c)
		if err != nil {
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

		if err := s.addUserToIAM(ctx, user, project); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusCreated, formatAADUser(user))
	})

	g.GET("/workspaces/:workspaceID/projects/:projectID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		project, err := s.validRequestURL(ctx, c)
		if err != nil {
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
		existInIAM, _, err := s.userInTheProject(ctx, user, project)
		if err != nil {
			return err
		}
		if !existInIAM {
			return c.JSON(http.StatusNotFound, map[string]any{
				"schemas": []string{
					"urn:ietf:params:scim:api:messages:2.0:Error",
				},
				"status": "404",
			})
		}

		return c.JSON(http.StatusOK, formatAADUser(user))
	})

	g.GET("/workspaces/:workspaceID/projects/:projectID/Users", func(c echo.Context) error {
		ctx := c.Request().Context()
		project, err := s.validRequestURL(ctx, c)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to find project, error %v", err))
		}

		// AAD SCIM will send ?filter=userName eq "{user name}" query
		filters, err := v1.ParseFilter(c.QueryParam("filter"))
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
			if expr.Operator != v1.ComparatorTypeEqual {
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
			existInIAM, _, err := s.userInTheProject(ctx, user, project)
			if err != nil {
				slog.Error("failed to check if user exist in iam", slog.String("user", user.Email), slog.String("project", project.ResourceID), log.BBError(err))
				continue
			}
			if !existInIAM {
				continue
			}
			response.TotalResults += 1
			response.Resources = append(response.Resources, formatAADUser(user))
		}

		return c.JSON(http.StatusOK, response)
	})

	g.DELETE("/workspaces/:workspaceID/projects/:projectID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		project, err := s.validRequestURL(ctx, c)
		if err != nil {
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

		if err := s.removeUserFromIAM(ctx, user, project); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusNoContent, "")
	})

	g.PATCH("/workspaces/:workspaceID/projects/:projectID/Users/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		project, err := s.validRequestURL(ctx, c)
		if err != nil {
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

		if delete := updateUser.Delete; delete != nil {
			if *delete {
				if err := s.removeUserFromIAM(ctx, user, project); err != nil {
					return c.String(http.StatusInternalServerError, err.Error())
				}
			} else {
				if err := s.addUserToIAM(ctx, user, project); err != nil {
					return c.String(http.StatusInternalServerError, err.Error())
				}
			}
			// Do not delete the user in the workspace. Only revoke him from the project.
			updateUser.Delete = nil
		}

		updatedUser, err := s.store.UpdateUser(ctx, user, updateUser, api.SystemBotID)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to update user, error %v", err))
		}

		return c.JSON(http.StatusOK, formatAADUser(updatedUser))
	})
}

func (s *Service) removeUserFromIAM(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage) error {
	policy, err := s.store.GetProjectIamPolicy(ctx, project.UID)
	if err != nil {
		return errors.Wrapf(err, "failed to get iam policy for project %s", project.ResourceID)
	}
	member := fmt.Sprintf("%s%s", common.UserBindingPrefix, user.Email)
	for _, binding := range policy.Policy.Bindings {
		index := slices.Index(binding.Members, member)
		if index >= 0 {
			binding.Members = slices.Delete(binding.Members, index, index+1)
		}
	}

	return s.updateIAMPolicy(ctx, project.UID, policy.Policy)
}

func (s *Service) userInTheProject(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage) (bool, *storepb.IamPolicy, error) {
	policy, err := s.store.GetProjectIamPolicy(ctx, project.UID)
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to get iam policy for project %s", project.ResourceID)
	}

	roles := utils.GetUserRolesInIamPolicy(ctx, s.store, user, policy.Policy)
	return len(roles) > 0, policy.Policy, nil
}

func (s *Service) addUserToIAM(ctx context.Context, user *store.UserMessage, project *store.ProjectMessage) error {
	existInIAM, policy, err := s.userInTheProject(ctx, user, project)
	if err != nil {
		return err
	}
	if !existInIAM {
		policy.Bindings = append(policy.Bindings, &storepb.Binding{
			Role: common.FormatRole(api.ProjectViewer.String()),
			Members: []string{
				fmt.Sprintf("%s%s", common.UserBindingPrefix, user.Email),
			},
		})

		return s.updateIAMPolicy(ctx, project.UID, policy)
	}

	return nil
}

func (s *Service) updateIAMPolicy(ctx context.Context, projectUID int, iamPolicy *storepb.IamPolicy) error {
	policyPayload, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal iam policy")
	}
	if _, err := s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		ResourceUID:       projectUID,
		ResourceType:      api.PolicyResourceTypeProject,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeIAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}, api.SystemBotID); err != nil {
		return errors.Wrapf(err, "failed to update iam policy")
	}
	return nil
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

func (s *Service) validRequestURL(ctx context.Context, c echo.Context) (*store.ProjectMessage, error) {
	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, err
	}
	if setting.ExternalUrl == "" {
		return nil, errors.Errorf("external URL is empty")
	}

	workspaceID := c.Param("workspaceID")
	projectID := c.Param("projectID")

	myWorkspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	if myWorkspaceID != workspaceID {
		return nil, errors.Errorf("invalid workspace id %q, my ID %q", workspaceID, myWorkspaceID)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, err
	}
	if project == nil || project.Deleted {
		return nil, errors.Errorf("project %q does not exist or has been deleted", projectID)
	}
	return project, nil
}
