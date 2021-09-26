package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerProjectMemberRoutes(g *echo.Group) {
	g.POST("/project/:projectId/member", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		projectMemberCreate := &api.ProjectMemberCreate{
			ProjectId: projectId,
			CreatorId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectMemberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project membership request").SetInternal(err)
		}

		projectMember, err := s.ProjectMemberService.CreateProjectMember(context.Background(), projectMemberCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, "User is already a project member")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project member").SetInternal(err)
		}

		if err := s.ComposeProjectMemberRelationship(context.Background(), projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created project membership relationship").SetInternal(err)
		}

		{
			activityCreate := &api.ActivityCreate{
				CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
				ContainerId: projectId,
				Type:        api.ActivityProjectMemberCreate,
				Level:       api.ACTIVITY_INFO,
				Comment: fmt.Sprintf("Granted %s to %s (%s).",
					projectMember.Principal.Name, projectMember.Principal.Email, projectMember.Role),
			}
			_, err = s.ActivityManager.CreateActivity(context.Background(), activityCreate, &ActivityMeta{})
			if err != nil {
				s.l.Warn("Failed to create project activity after creating member",
					zap.Int("project_id", projectId),
					zap.Int("principal_id", projectMember.Principal.ID),
					zap.String("principal_name", projectMember.Principal.Name),
					zap.String("role", projectMember.Role),
					zap.Error(err))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create projectMember response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectId/member/:memberId", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("memberId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberId"))).SetInternal(err)
		}

		existingProjectMember, err := s.ProjectMemberService.FindProjectMember(context.Background(), &api.ProjectMemberFind{ID: &id})
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}

		projectMemberPatch := &api.ProjectMemberPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectMemberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change project membership").SetInternal(err)
		}

		projectMember, err := s.ProjectMemberService.PatchProjectMember(context.Background(), projectMemberPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change project membership ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeProjectMemberRelationship(context.Background(), projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated project membership relationship").SetInternal(err)
		}

		{
			activityCreate := &api.ActivityCreate{
				CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
				ContainerId: projectId,
				Type:        api.ActivityProjectMemberRoleUpdate,
				Level:       api.ACTIVITY_INFO,
				Comment: fmt.Sprintf("Changed %s (%s) from %s to %s.",
					projectMember.Principal.Name, projectMember.Principal.Email, existingProjectMember.Role, projectMember.Role),
			}
			_, err = s.ActivityManager.CreateActivity(context.Background(), activityCreate, &ActivityMeta{})
			if err != nil {
				s.l.Warn("Failed to create project activity after updating member role",
					zap.Int("project_id", projectId),
					zap.Int("principal_id", projectMember.Principal.ID),
					zap.String("principal_name", projectMember.Principal.Name),
					zap.String("old_role", existingProjectMember.Role),
					zap.String("new_role", projectMember.Role),
					zap.Error(err))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project membership change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/project/:projectId/member/:memberId", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("memberId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberId"))).SetInternal(err)
		}

		projectMember, err := s.ProjectMemberService.FindProjectMember(context.Background(), &api.ProjectMemberFind{ID: &id})
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}

		projectMemberDelete := &api.ProjectMemberDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.ProjectMemberService.DeleteProjectMember(context.Background(), projectMemberDelete)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}

		{
			projectMember.Principal, err = s.ComposePrincipalById(context.Background(), projectMember.PrincipalId)
			if err == nil {
				activityCreate := &api.ActivityCreate{
					CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
					ContainerId: projectId,
					Type:        api.ActivityProjectMemberDelete,
					Level:       api.ACTIVITY_INFO,
					Comment: fmt.Sprintf("Revoked %s from %s (%s).",
						projectMember.Role, projectMember.Principal.Name, projectMember.Principal.Email),
				}
				_, err = s.ActivityManager.CreateActivity(context.Background(), activityCreate, &ActivityMeta{})
			}
			if err != nil {
				s.l.Warn("Failed to create project activity after deleting member",
					zap.Int("project_id", projectId),
					zap.Int("principal_id", projectMember.Principal.ID),
					zap.String("principal_name", projectMember.Principal.Name),
					zap.String("role", projectMember.Role),
					zap.Error(err))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) ComposeProjectMemberListByProjectId(ctx context.Context, projectId int) ([]*api.ProjectMember, error) {
	projectMemberFind := &api.ProjectMemberFind{
		ProjectId: &projectId,
	}
	projectMemberList, err := s.ProjectMemberService.FindProjectMemberList(ctx, projectMemberFind)
	if err != nil {
		return nil, err
	}

	for _, projectMember := range projectMemberList {
		if err := s.ComposeProjectMemberRelationship(ctx, projectMember); err != nil {
			return nil, err
		}
	}
	return projectMemberList, nil
}

func (s *Server) ComposeProjectMemberRelationship(ctx context.Context, projectMember *api.ProjectMember) error {
	var err error

	projectMember.Creator, err = s.ComposePrincipalById(context.Background(), projectMember.CreatorId)
	if err != nil {
		return err
	}

	projectMember.Updater, err = s.ComposePrincipalById(context.Background(), projectMember.UpdaterId)
	if err != nil {
		return err
	}

	projectMember.Principal, err = s.ComposePrincipalById(context.Background(), projectMember.PrincipalId)
	if err != nil {
		return err
	}

	return nil
}
