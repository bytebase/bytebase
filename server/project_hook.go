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

func (s *Server) registerProjectHookRoutes(g *echo.Group) {
	g.GET("/project/:projectId/hook", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		find := &api.ProjectHookFind{
			ProjectId: &projectId,
		}
		list, err := s.ProjectHookService.FindProjectHookList(context.Background(), find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch webhook list for project ID: %d", projectId)).SetInternal(err)
		}

		for _, hook := range list {
			if err := s.ComposeProjectHookRelationship(context.Background(), hook); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch webhook relationship: %v", hook.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook response: %v", projectId)).SetInternal(err)
		}
		return nil
	})

	g.POST("/project/:projectId/hook", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		hookCreate := &api.ProjectHookCreate{
			CreatorId: c.Get(GetPrincipalIdContextKey()).(int),
			ProjectId: projectId,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, hookCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project webhook request").SetInternal(err)
		}

		hook, err := s.ProjectHookService.CreateProjectHook(context.Background(), hookCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Webhook url already exists in the project: %s", hook.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project webhook").SetInternal(err)
		}

		if err := s.ComposeProjectHookRelationship(context.Background(), hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create project webhook response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectId/hook/:hookId", func(c echo.Context) error {
		_, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("hookId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Webhook ID is not a number: %s", c.Param("hookId"))).SetInternal(err)
		}

		hookPatch := &api.ProjectHookPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, hookPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change project webhook").SetInternal(err)
		}

		hook, err := s.ProjectHookService.PatchProjectHook(context.Background(), hookPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
			}
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Webhook url already exists in the project: %s", *hookPatch.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change project webhook ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeProjectHookRelationship(context.Background(), hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated project webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/project/:projectId/hook/:hookId", func(c echo.Context) error {
		_, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("hookId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Webhook ID is not a number: %s", c.Param("hookId"))).SetInternal(err)
		}

		hookDelete := &api.ProjectHookDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.ProjectHookService.DeleteProjectHook(context.Background(), hookDelete)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project webhook ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) ComposeProjectHookRelationship(ctx context.Context, hook *api.ProjectHook) error {
	var err error

	hook.Creator, err = s.ComposePrincipalById(context.Background(), hook.CreatorId)
	if err != nil {
		return err
	}

	hook.Updater, err = s.ComposePrincipalById(context.Background(), hook.UpdaterId)
	if err != nil {
		return err
	}

	return nil
}
