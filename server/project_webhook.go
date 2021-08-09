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

func (s *Server) registerProjectWebhookRoutes(g *echo.Group) {
	g.GET("/project/:projectId/webhook", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		find := &api.ProjectWebhookFind{
			ProjectId: &projectId,
		}
		list, err := s.ProjectWebhookService.FindProjectWebhookList(context.Background(), find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch webhook list for project ID: %d", projectId)).SetInternal(err)
		}

		for _, hook := range list {
			if err := s.ComposeProjectWebhookRelationship(context.Background(), hook); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch webhook relationship: %v", hook.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook response: %v", projectId)).SetInternal(err)
		}
		return nil
	})

	g.POST("/project/:projectId/webhook", func(c echo.Context) error {
		projectId, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		hookCreate := &api.ProjectWebhookCreate{
			CreatorId: c.Get(GetPrincipalIdContextKey()).(int),
			ProjectId: projectId,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, hookCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project webhook request").SetInternal(err)
		}

		hook, err := s.ProjectWebhookService.CreateProjectWebhook(context.Background(), hookCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Webhook url already exists in the project: %s", hookCreate.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project webhook").SetInternal(err)
		}

		if err := s.ComposeProjectWebhookRelationship(context.Background(), hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create project webhook response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:projectId/webhook/:hookId", func(c echo.Context) error {
		_, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("hookId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project webhook ID is not a number: %s", c.Param("hookId"))).SetInternal(err)
		}

		find := &api.ProjectWebhookFind{
			ID: &id,
		}
		hook, err := s.ProjectWebhookService.FindProjectWebhook(context.Background(), find)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project webhook ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeProjectWebhookRelationship(context.Background(), hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectId/webhook/:hookId", func(c echo.Context) error {
		_, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("hookId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project webhook ID is not a number: %s", c.Param("hookId"))).SetInternal(err)
		}

		hookPatch := &api.ProjectWebhookPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, hookPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change project webhook").SetInternal(err)
		}

		hook, err := s.ProjectWebhookService.PatchProjectWebhook(context.Background(), hookPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
			}
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Hook url already exists in the project: %s", *hookPatch.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change project webhook ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeProjectWebhookRelationship(context.Background(), hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated project webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, hook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/project/:projectId/webhook/:hookId", func(c echo.Context) error {
		_, err := strconv.Atoi(c.Param("projectId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectId"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("hookId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Webhook ID is not a number: %s", c.Param("hookId"))).SetInternal(err)
		}

		hookDelete := &api.ProjectWebhookDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.ProjectWebhookService.DeleteProjectWebhook(context.Background(), hookDelete)
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

func (s *Server) ComposeProjectWebhookRelationship(ctx context.Context, hook *api.ProjectWebhook) error {
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
