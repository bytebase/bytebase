package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) registerSheetOrganizerRoutes(g *echo.Group) {
	g.PATCH("/sheet/:sheetID/organizer", func(c echo.Context) error {
		ctx := c.Request().Context()
		sheetID, err := strconv.Atoi(c.Param("sheetID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Sheet ID is not a number: %s", c.Param("sheetID"))).SetInternal(err)
		}

		principalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetOrganizerUpsert := &api.SheetOrganizerUpsert{
			SheetID:     sheetID,
			PrincipalID: principalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetOrganizerUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch sheet organizer request").SetInternal(err)
		}

		if _, err := s.store.UpsertSheetOrganizer(ctx, sheetOrganizerUpsert); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to upsert organizer %d to sheet %d", principalID, sheetID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}
