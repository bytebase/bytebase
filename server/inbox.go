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
)

func (s *Server) registerInboxRoutes(g *echo.Group) {
	g.GET("/inbox", func(c echo.Context) error {
		ctx := context.Background()
		inboxFind := &api.InboxFind{}
		userIDStr := c.QueryParams().Get("user")
		if userIDStr != "" {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", userIDStr)).SetInternal(err)
			}
			inboxFind.ReceiverID = &userID
		}
		createdAfterStr := c.QueryParams().Get("created")
		if createdAfterStr != "" {
			createdTs, err := strconv.ParseInt(createdAfterStr, 10, 64)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter created is not a number: %s", createdAfterStr)).SetInternal(err)
			}
			inboxFind.ReadCreatedAfterTs = &createdTs
		}
		list, err := s.InboxService.FindInboxList(ctx, inboxFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch inbox list").SetInternal(err)
		}

		for _, inbox := range list {
			if err := s.composeActivityRelationship(ctx, inbox.Activity); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch inbox activity relationship: %v", inbox.Activity.ID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal inbox list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/inbox/summary", func(c echo.Context) error {
		ctx := context.Background()
		userIDStr := c.QueryParams().Get("user")
		if userIDStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing query parameter user")
		}
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", userIDStr)).SetInternal(err)
		}

		summary, err := s.InboxService.FindInboxSummary(ctx, userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch inbox summary for user ID: %d", userID)).SetInternal(err)
		}

		return c.JSON(http.StatusOK, summary)
	})

	g.PATCH("/inbox/:inboxID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("inboxID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("inboxID"))).SetInternal(err)
		}

		inboxPatch := &api.InboxPatch{
			ID: id,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, inboxPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch inbox request").SetInternal(err)
		}

		inbox, err := s.InboxService.PatchInbox(ctx, inboxPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Inbox ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch inbox ID: %v", id)).SetInternal(err)
		}

		if err := s.composeActivityRelationship(ctx, inbox.Activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated inbox activity relationship: %v", inbox.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, inbox); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal inbox ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}
