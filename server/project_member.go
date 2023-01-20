package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/store"
)

func (s *Server) registerProjectMemberRoutes(g *echo.Group) {
	g.POST("/project/:projectID/member", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		projectMemberCreate := &api.ProjectMemberCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectMemberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create project membership request").SetInternal(err)
		}
		creatorID := c.Get(getPrincipalIDContextKey()).(int)

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return err
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return err
		}
		user, err := s.store.GetUserByID(ctx, projectMemberCreate.PrincipalID)
		if err != nil {
			return err
		}
		if user == nil {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}

		newPolicy := removeMember(policy, user)
		foundRole := false
		for _, binding := range newPolicy.Bindings {
			if binding.Role == api.Role(projectMemberCreate.Role) {
				binding.Members = append(binding.Members, user)
				foundRole = true
				break
			}
		}
		if !foundRole {
			newPolicy.Bindings = append(newPolicy.Bindings, &store.PolicyBinding{
				Role:    api.Role(projectMemberCreate.Role),
				Members: []*store.UserMessage{user},
			})
		}
		if _, err := s.store.SetProjectIAMPolicy(ctx, newPolicy, creatorID, project.UID); err != nil {
			return err
		}

		activityCreate := &api.ActivityCreate{
			CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
			ContainerID: projectID,
			Type:        api.ActivityProjectMemberCreate,
			Level:       api.ActivityInfo,
			Comment:     fmt.Sprintf("Granted %s to %s (%s).", user.Name, user.Email, projectMemberCreate.Role),
		}
		if _, err := s.store.CreateActivity(ctx, activityCreate); err != nil {
			log.Warn("Failed to create project activity after creating member",
				zap.Int("project_id", projectID),
				zap.Int("principal_id", user.ID),
				zap.String("principal_name", user.Name),
				zap.String("role", string(projectMemberCreate.Role)),
				zap.Error(err))
		}

		principal, err := s.store.GetPrincipalByID(ctx, user.ID)
		if err != nil {
			return err
		}
		composedProjectMember := &api.ProjectMember{
			ID:        user.ID,
			ProjectID: project.UID,
			Role:      string(projectMemberCreate.Role),
			Principal: principal,
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedProjectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create projectMember response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectID/member/:memberID", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		principalID, err := strconv.Atoi(c.Param("memberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberID"))).SetInternal(err)
		}
		projectMemberPatch := &api.ProjectMemberPatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectMemberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed change project membership").SetInternal(err)
		}
		if projectMemberPatch.Role == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "role is required")
		}
		newRole := *projectMemberPatch.Role
		updaterID := c.Get(getPrincipalIDContextKey()).(int)

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return err
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return err
		}
		user, err := s.store.GetUserByID(ctx, principalID)
		if err != nil {
			return err
		}
		if user == nil {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}

		newPolicy := removeMember(policy, user)
		foundRole := false
		for _, binding := range newPolicy.Bindings {
			if binding.Role == api.Role(newRole) {
				binding.Members = append(binding.Members, user)
				foundRole = true
				break
			}
		}
		if !foundRole {
			newPolicy.Bindings = append(newPolicy.Bindings, &store.PolicyBinding{
				Role:    api.Role(newRole),
				Members: []*store.UserMessage{user},
			})
		}
		if _, err := s.store.SetProjectIAMPolicy(ctx, newPolicy, updaterID, project.UID); err != nil {
			return err
		}

		activityCreate := &api.ActivityCreate{
			CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
			ContainerID: projectID,
			Type:        api.ActivityProjectMemberRoleUpdate,
			Level:       api.ActivityInfo,
			Comment:     fmt.Sprintf("Changed %s (%s) to %s.", user.Name, user.Email, newRole),
		}
		if _, err := s.store.CreateActivity(ctx, activityCreate); err != nil {
			log.Warn("Failed to create project activity after updating member role",
				zap.Int("project_id", projectID),
				zap.Int("principal_id", user.ID),
				zap.String("principal_name", user.Name),
				zap.String("new_role", newRole),
				zap.Error(err))
		}

		principal, err := s.store.GetPrincipalByID(ctx, user.ID)
		if err != nil {
			return err
		}
		composedProjectMember := &api.ProjectMember{
			ID:        user.ID,
			ProjectID: project.UID,
			Role:      string(newRole),
			Principal: principal,
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedProjectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project membership change response: %v", principalID)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/project/:projectID/member/:memberID", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		principalID, err := strconv.Atoi(c.Param("memberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberID"))).SetInternal(err)
		}
		updaterID := c.Get(getPrincipalIDContextKey()).(int)

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
		if err != nil {
			return err
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, "project not found")
		}
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return err
		}
		user, err := s.store.GetUserByID(ctx, principalID)
		if err != nil {
			return err
		}
		if user == nil {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}

		newPolicy := removeMember(policy, user)
		if _, err := s.store.SetProjectIAMPolicy(ctx, newPolicy, updaterID, project.UID); err != nil {
			return err
		}

		activityCreate := &api.ActivityCreate{
			CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
			ContainerID: projectID,
			Type:        api.ActivityProjectMemberDelete,
			Level:       api.ActivityInfo,
			Comment:     fmt.Sprintf("Revoked %s (%s).", user.Name, user.Email),
		}
		if _, err := s.store.CreateActivity(ctx, activityCreate); err != nil {
			log.Warn("Failed to create project activity after deleting member",
				zap.Int("project_id", projectID),
				zap.Int("principal_id", user.ID),
				zap.String("principal_name", user.Name),
				zap.Error(err))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func removeMember(policy *store.IAMPolicyMessage, user *store.UserMessage) *store.IAMPolicyMessage {
	newPolicy := &store.IAMPolicyMessage{}
	for _, binding := range policy.Bindings {
		newBinding := &store.PolicyBinding{Role: binding.Role}
		for _, member := range binding.Members {
			if member.ID == user.ID {
				continue
			}
			newBinding.Members = append(newBinding.Members, member)
		}
		newPolicy.Bindings = append(newPolicy.Bindings, newBinding)
	}
	return newPolicy
}
