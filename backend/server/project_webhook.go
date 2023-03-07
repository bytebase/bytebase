package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/gosimple/slug"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	webhookPlugin "github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerProjectWebhookRoutes(g *echo.Group) {
	g.GET("/project/:projectID/webhook", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		webhookList, err := s.store.FindProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
			ProjectID: &projectID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch webhook list for project ID: %d", projectID)).SetInternal(err)
		}

		var apiWebhookList []*api.ProjectWebhook
		for _, webhook := range webhookList {
			apiWebhookList = append(apiWebhookList, webhook.ToAPIProjectWebhook())
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, apiWebhookList); err != nil {
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create project webhook request").SetInternal(err)
		}
		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %v", projectID))
		}

		webhook, err := s.store.CreateProjectWebhookV2(ctx, c.Get(getPrincipalIDContextKey()).(int), projectID, project.ResourceID, &store.ProjectWebhookMessage{})
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Webhook url already exists in the project: %s", hookCreate.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project webhook").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, webhook.ToAPIProjectWebhook()); err != nil {
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

		webhook, err := s.store.GetProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
			ID: &id,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project webhook ID: %v", id)).SetInternal(err)
		}
		if webhook == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, webhook.ToAPIProjectWebhook()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectID/webhook/:webhookID", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("webhookID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project webhook ID is not a number: %s", c.Param("webhookID"))).SetInternal(err)
		}

		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %v", projectID))
		}

		hookPatch := &api.ProjectWebhookPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, hookPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed change project webhook").SetInternal(err)
		}
		var activityList []string
		if hookPatch.ActivityList != nil {
			activityList = strings.Split(*hookPatch.ActivityList, ",")
		}

		webhook, err := s.store.UpdateProjectWebhookV2(ctx, c.Get(getPrincipalIDContextKey()).(int), project.ID, project.ResourceID, id, &store.UpdateProjectWebhookMessage{
			Title:        hookPatch.Name,
			URL:          hookPatch.URL,
			ActivityList: activityList,
		})
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
			}
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Hook url already exists in the project: %s", *hookPatch.URL))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change project webhook ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, webhook.ToAPIProjectWebhook()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project webhook change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/project/:projectID/webhook/:webhookID", func(c echo.Context) error {
		ctx := c.Request().Context()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("webhookID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Webhook ID is not a number: %s", c.Param("webhookID"))).SetInternal(err)
		}

		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", projectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %v", projectID))
		}
		if err := s.store.DeleteProjectWebhookV2(ctx, project.ID, project.ResourceID, id); err != nil {
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

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &projectID})
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

		webhook, err := s.store.GetProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
			ID: &id,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project webhook ID: %v", id)).SetInternal(err)
		}
		if webhook == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project webhook ID not found: %d", id))
		}

		setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find workspace setting").SetInternal(err)
		}

		result := &api.ProjectWebhookTestResult{}
		err = webhookPlugin.Post(
			webhook.Type,
			webhookPlugin.Context{
				URL:          webhook.URL,
				Level:        webhookPlugin.WebhookInfo,
				ActivityType: string(api.ActivityIssueCreate),
				Title:        fmt.Sprintf("Test webhook %q", webhook.Title),
				Description:  "This is a test",
				Link:         fmt.Sprintf("%s/project/%s/webhook/%s", setting.ExternalUrl, getProjectSlug(project), api.ProjectWebhookSlug(webhook.Title, webhook.ID)),
				CreatorID:    api.SystemBotID,
				CreatorName:  "Bytebase",
				CreatorEmail: "support@bytebase.com",
				CreatedTs:    time.Now().Unix(),
				Project:      &webhookPlugin.Project{Name: project.Title},
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

// getProjectSlug is the slug formatter for Project.
func getProjectSlug(project *store.ProjectMessage) string {
	return fmt.Sprintf("%s-%d", slug.Make(project.Title), project.UID)
}
