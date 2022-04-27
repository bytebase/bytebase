package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	webhookPlugin "github.com/bytebase/bytebase/plugin/webhook"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerProjectWebhookRoutes(g *echo.Group) {
	g.GET("/project/:projectID/webhook", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		find := &api.ProjectWebhookFind{
			ProjectID: &projectID,
		}
		webhookRawList, err := s.ProjectWebhookService.FindProjectWebhookList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch webhook list for project ID: %d", projectID)).SetInternal(err)
		}
		var webhookList []*api.ProjectWebhook
		for _, raw := range webhookRawList {
			webhook, err := s.composeProjectWebhookRelationship(ctx, raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch webhook relationship: %v", raw.Name)).SetInternal(err)
			}
			webhookList = append(webhookList, webhook)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, webhookList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook response: %v", projectID)).SetInternal(err)
		}
		return nil
	})

	g.POST("/project/:projectID/webhook", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		hookCreate := &api.ProjectWebhookCreate{
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
			ProjectID: projectID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, hookCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project webhook request").SetInternal(err)
		}

		webhookRaw, err := s.ProjectWebhookService.CreateProjectWebhook(ctx, hookCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Webhook url already exists in the project: %s", hookCreate.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project webhook").SetInternal(err)
		}

		webhook, err := s.composeProjectWebhookRelationship(ctx, webhookRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, webhook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create project webhook response").SetInternal(err)
		}
		return nil
	})

	g.GET("/project/:projectID/webhook/:webhookID", func(c echo.Context) error {
		ctx := c.Request().Context()
		_, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("webhookID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project webhook ID is not a number: %s", c.Param("webhookID"))).SetInternal(err)
		}

		find := &api.ProjectWebhookFind{
			ID: &id,
		}
		webhookRaw, err := s.ProjectWebhookService.FindProjectWebhook(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project webhook ID: %v", id)).SetInternal(err)
		}
		if webhookRaw == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
		}

		webhook, err := s.composeProjectWebhookRelationship(ctx, webhookRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, webhook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectID/webhook/:webhookID", func(c echo.Context) error {
		ctx := c.Request().Context()
		_, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("webhookID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project webhook ID is not a number: %s", c.Param("webhookID"))).SetInternal(err)
		}

		hookPatch := &api.ProjectWebhookPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, hookPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change project webhook").SetInternal(err)
		}

		webhookRaw, err := s.ProjectWebhookService.PatchProjectWebhook(ctx, hookPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
			}
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Hook url already exists in the project: %s", *hookPatch.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change project webhook ID: %v", id)).SetInternal(err)
		}

		webhook, err := s.composeProjectWebhookRelationship(ctx, webhookRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated project webhook relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, webhook); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/project/:projectID/webhook/:webhookID", func(c echo.Context) error {
		ctx := c.Request().Context()
		_, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("webhookID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Webhook ID is not a number: %s", c.Param("webhookID"))).SetInternal(err)
		}

		hookDelete := &api.ProjectWebhookDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.ProjectWebhookService.DeleteProjectWebhook(ctx, hookDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project webhook ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.GET("/project/:projectID/webhook/:webhookID/test", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", projectID))
		}

		id, err := strconv.Atoi(c.Param("webhookID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project webhook ID is not a number: %s", c.Param("webhookID"))).SetInternal(err)
		}

		find := &api.ProjectWebhookFind{
			ID: &id,
		}
		webhookRaw, err := s.ProjectWebhookService.FindProjectWebhook(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project webhook ID: %v", id)).SetInternal(err)
		}
		if webhookRaw == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
		}
		webhook, err := s.composeProjectWebhookRelationship(ctx, webhookRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose project webhook relationship with ID %d", id)).SetInternal(err)
		}

		result := &api.ProjectWebhookTestResult{}
		err = webhookPlugin.Post(
			webhook.Type,
			webhookPlugin.Context{
				URL:          webhook.URL,
				Level:        webhookPlugin.WebhookInfo,
				Title:        fmt.Sprintf("Test webhook %q", webhook.Name),
				Description:  "This is a test",
				Link:         fmt.Sprintf("%s:%d/project/%s/webhook/%s", s.frontendHost, s.frontendPort, api.ProjectSlug(project), api.ProjectWebhookSlug(webhook)),
				CreatorName:  "Bytebase",
				CreatorEmail: "support@bytebase.com",
				CreatedTs:    time.Now().Unix(),
				MetaList: []webhookPlugin.Meta{
					{
						Name:  "Project",
						Value: project.Name,
					},
				},
			},
		)

		if err != nil {
			result.Error = err.Error()
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, result); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook response: %v", projectID)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeProjectWebhookRelationship(ctx context.Context, raw *api.ProjectWebhookRaw) (*api.ProjectWebhook, error) {
	webhook := raw.ToProjectWebhook()

	creator, err := s.store.GetPrincipalByID(ctx, webhook.CreatorID)
	if err != nil {
		return nil, err
	}
	webhook.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, webhook.UpdaterID)
	if err != nil {
		return nil, err
	}
	webhook.Updater = updater

	return webhook, nil
}
