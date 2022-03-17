package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerActivityRoutes(g *echo.Group) {
	g.POST("/activity", func(c echo.Context) error {
		ctx := context.Background()
		activityCreate := &api.ActivityCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, activityCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create activity request").SetInternal(err)
		}

		activityCreate.Level = api.ActivityInfo
		activityCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		var foundIssue *api.Issue
		if activityCreate.Type == api.ActivityIssueCommentCreate {
			issueFind := &api.IssueFind{
				ID: &activityCreate.ContainerID,
			}
			issue, err := s.IssueService.FindIssue(ctx, issueFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when creating the comment: %d", activityCreate.ContainerID)).SetInternal(err)
			}
			if issue == nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unable to find issue ID for creating the comment: %d", activityCreate.ContainerID))
			}

			bytes, err := json.Marshal(api.ActivityIssueCommentCreatePayload{
				IssueName: issue.Name,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
			}
			activityCreate.Payload = string(bytes)
			foundIssue = issue
		}

		activity, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{
			issue: foundIssue,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal created activity response").SetInternal(err)
		}
		return nil
	})

	g.GET("/activity", func(c echo.Context) error {
		ctx := context.Background()
		activityFind := &api.ActivityFind{}
		if creatorIDStr := c.QueryParams().Get("user"); creatorIDStr != "" {
			creatorID, err := strconv.Atoi(creatorIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", creatorIDStr)).SetInternal(err)
			}
			activityFind.CreatorID = &creatorID
		}
		if typeStr := c.QueryParams().Get("type"); typeStr != "" {
			activityFind.Type = &typeStr
		}
		if levelStr := c.QueryParams().Get("level"); levelStr != "" {
			activityLevel := api.ActivityLevel(levelStr)
			activityFind.Level = &activityLevel
		}
		if containerIDStr := c.QueryParams().Get("container"); containerIDStr != "" {
			containerID, err := strconv.Atoi(containerIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter container is not a number: %s", containerIDStr)).SetInternal(err)
			}
			activityFind.ContainerID = &containerID
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter limit is not a number: %s", limitStr)).SetInternal(err)
			}
			activityFind.Limit = &limit
		}
		activityRawList, err := s.ActivityService.FindActivityList(ctx, activityFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch activity list").SetInternal(err)
		}

		var activityList []*api.Activity
		for _, activityRaw := range activityRawList {
			activity, err := s.composeActivityRelationship(ctx, activityRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch activity relationship: %v", activityRaw.ID)).SetInternal(err)
			}
			activityList = append(activityList, activity)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activityList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal activity list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/activity/:activityID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("activityID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("activityID"))).SetInternal(err)
		}

		activityPatch := &api.ActivityPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, activityPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch activity request").SetInternal(err)
		}

		activityRaw, err := s.ActivityService.PatchActivity(ctx, activityPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Activity ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch activity ID: %v", id)).SetInternal(err)
		}

		activity, err := s.composeActivityRelationship(ctx, activityRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated activity relationship: %v", activityRaw.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/activity/:activityID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("activityID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("activityID"))).SetInternal(err)
		}

		activityDelete := &api.ActivityDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.ActivityService.DeleteActivity(ctx, activityDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete activity ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) composeActivityRelationship(ctx context.Context, raw *api.ActivityRaw) (*api.Activity, error) {
	activity := raw.ToActivity()

	creator, err := s.composePrincipalByID(ctx, activity.CreatorID)
	if err != nil {
		return nil, err
	}
	activity.Creator = creator

	updater, err := s.composePrincipalByID(ctx, activity.UpdaterID)
	if err != nil {
		return nil, err
	}
	activity.Updater = updater

	return activity, nil
}
