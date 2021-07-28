package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerActivityRoutes(g *echo.Group) {
	g.POST("/activity", func(c echo.Context) error {
		activityCreate := &api.ActivityCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, activityCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create activity request").SetInternal(err)
		}

		activityCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		if activityCreate.Type == api.ActivityIssueCommentCreate {
			issueFind := &api.IssueFind{
				ID: &activityCreate.ContainerId,
			}
			issue, err := s.IssueService.FindIssue(context.Background(), issueFind)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unable to find issue ID for creating the comment: %d", activityCreate.ContainerId))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when creating the comment: %d", activityCreate.ContainerId)).SetInternal(err)
			}

			bytes, err := json.Marshal(api.ActivityIssueCommentCreatePayload{
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct comment payload").SetInternal(err)
			}
			activityCreate.Payload = string(bytes)
		}

		activity, err := s.ActivityService.CreateActivity(context.Background(), activityCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity").SetInternal(err)
		}

		if err := s.ComposeActivityRelationship(context.Background(), activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created activity relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal created activity response").SetInternal(err)
		}
		return nil
	})

	g.GET("/activity", func(c echo.Context) error {
		activityFind := &api.ActivityFind{}
		containerIdStr := c.QueryParams().Get("container")
		if containerIdStr != "" {
			containerId, err := strconv.Atoi(containerIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter container is not a number: %s", containerIdStr)).SetInternal(err)
			}
			activityFind.ContainerId = &containerId
		}
		list, err := s.ActivityService.FindActivityList(context.Background(), activityFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch activity list").SetInternal(err)
		}

		for _, activity := range list {
			if err := s.ComposeActivityRelationship(context.Background(), activity); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch activity relationship: %v", activity.ID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal activity list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/activity/:activityId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("activityId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("activityId"))).SetInternal(err)
		}

		activityPatch := &api.ActivityPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, activityPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch activity request").SetInternal(err)
		}

		activity, err := s.ActivityService.PatchActivity(context.Background(), activityPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Activity ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch activity ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeActivityRelationship(context.Background(), activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated activity relationship: %v", activity.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/activity/:activityId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("activityId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("activityId"))).SetInternal(err)
		}

		activityDelete := &api.ActivityDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.ActivityService.DeleteActivity(context.Background(), activityDelete)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Activity ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete activity ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) ComposeActivityRelationship(ctx context.Context, activity *api.Activity) error {
	var err error

	activity.Creator, err = s.ComposePrincipalById(context.Background(), activity.CreatorId)
	if err != nil {
		return err
	}

	activity.Updater, err = s.ComposePrincipalById(context.Background(), activity.UpdaterId)
	if err != nil {
		return err
	}

	return nil
}
