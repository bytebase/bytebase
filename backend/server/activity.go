package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerActivityRoutes(g *echo.Group) {
	g.POST("/activity", func(c echo.Context) error {
		ctx := c.Request().Context()
		activityCreate := &api.ActivityCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, activityCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create activity request").SetInternal(err)
		}
		if activityCreate.Type != api.ActivityIssueCommentCreate {
			return echo.NewHTTPError(http.StatusBadRequest, "Only allow to create activity for issue comment")
		}

		activityCreate.Level = api.ActivityInfo
		activityCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &activityCreate.ContainerID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch issue ID when creating the comment: %d", activityCreate.ContainerID)).SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unable to find issue ID for creating the comment: %d", activityCreate.ContainerID))
		}

		var payload api.ActivityIssueCommentCreatePayload
		if activityCreate.Payload != "" {
			if err := json.Unmarshal([]byte(activityCreate.Payload), &payload); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Failed to unmarshal payload %v", activityCreate.Payload).SetInternal(err)
			}
		}
		payload.IssueName = issue.Title
		bytes, err := json.Marshal(payload)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
		}
		activityCreate.Payload = string(bytes)

		activity, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal created activity response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/activity/:activityID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("activityID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("activityID"))).SetInternal(err)
		}

		activityPatch := &api.ActivityPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, activityPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch activity request").SetInternal(err)
		}
		// We only allow to update comment from frontend.
		activity, err := s.store.PatchActivity(ctx, activityPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Activity ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch activity ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}
