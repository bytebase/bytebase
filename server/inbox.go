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
		inboxFind := &api.InboxFind{}
		userIdStr := c.QueryParams().Get("user")
		if userIdStr != "" {
			userId, err := strconv.Atoi(userIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", userIdStr)).SetInternal(err)
			}
			inboxFind.ReceiverId = &userId
		}
		createdAfterStr := c.QueryParams().Get("created")
		if createdAfterStr != "" {
			createdTs, err := strconv.ParseInt(createdAfterStr, 10, 64)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter created is not a number: %s", createdAfterStr)).SetInternal(err)
			}
			inboxFind.ReadCreatedAfterTs = &createdTs
		}
		list, err := s.InboxService.FindInboxList(context.Background(), inboxFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch inbox list").SetInternal(err)
		}

		for _, inbox := range list {
			if err := s.ComposeActivityRelationship(context.Background(), inbox.Activity); err != nil {
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
		userIdStr := c.QueryParams().Get("user")
		if userIdStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing query parameter user")
		}
		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter user is not a number: %s", userIdStr)).SetInternal(err)
		}

		summary, err := s.InboxService.FindInboxSummary(context.Background(), userId)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch inbox summary for user ID: %d", userId)).SetInternal(err)
		}

		return c.JSON(http.StatusOK, summary)
	})

	g.PATCH("/inbox/:inboxId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("inboxId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("inboxId"))).SetInternal(err)
		}

		inboxPatch := &api.InboxPatch{
			ID: id,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, inboxPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch inbox request").SetInternal(err)
		}

		inbox, err := s.InboxService.PatchInbox(context.Background(), inboxPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Inbox ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch inbox ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeActivityRelationship(context.Background(), inbox.Activity); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated inbox activity relationship: %v", inbox.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, inbox); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal inbox ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}
