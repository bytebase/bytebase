package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerProjectRoutes(g *echo.Group) {
	g.POST("/project", func(c echo.Context) error {
		projectCreate := &api.ProjectCreate{WorkspaceId: api.DEFAULT_WORKPSACE_ID}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project request").SetInternal(err)
		}
		projectCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		project, err := s.ProjectService.CreateProject(context.Background(), projectCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Project name already exists: %s", projectCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project").SetInternal(err)
		}

		if err := s.ComposeProjectRelationship(context.Background(), project, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create project response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project", func(c echo.Context) error {
		workspaceId := api.DEFAULT_WORKPSACE_ID
		projectFind := &api.ProjectFind{
			WorkspaceId: &workspaceId,
		}
		if userIdStr := c.QueryParam("user"); userIdStr != "" {
			userId, err := strconv.Atoi(userIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", userIdStr)).SetInternal(err)
			}
			projectFind.PrincipalId = &userId
		}
		list, err := s.ProjectService.FindProjectList(context.Background(), projectFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch project list").SetInternal(err)
		}

		for _, project := range list {
			if err := s.ComposeProjectRelationship(context.Background(), project, c.Get(getIncludeKey()).([]string)); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal project list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:projectId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		project, err := s.ComposeProjectlById(context.Background(), id, c.Get(getIncludeKey()).([]string))
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		projectPatch := &api.ProjectPatch{
			ID:          id,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch project request").SetInternal(err)
		}

		project, err := s.ProjectService.PatchProject(context.Background(), projectPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch project ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeProjectRelationship(context.Background(), project, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, project); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeProjectlById(ctx context.Context, id int, includeList []string) (*api.Project, error) {
	projectFind := &api.ProjectFind{
		ID: &id,
	}
	project, err := s.ProjectService.FindProject(ctx, projectFind)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
			return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", id))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", id)).SetInternal(err)
	}

	if err := s.ComposeProjectRelationship(ctx, project, includeList); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Server) ComposeProjectRelationship(ctx context.Context, project *api.Project, includeList []string) error {
	var err error

	project.Creator, err = s.ComposePrincipalById(context.Background(), project.CreatorId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch creator for project: %v", project.Name)).SetInternal(err)
	}

	project.Updater, err = s.ComposePrincipalById(context.Background(), project.UpdaterId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updater for project: %v", project.Name)).SetInternal(err)
	}

	project.ProjectMemberList, err = s.ComposeProjectMemberListByProjectId(ctx, project.ID, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project member for project: %v", project.Name)).SetInternal(err)
	}

	return nil
}
