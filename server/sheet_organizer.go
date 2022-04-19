package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerSheetOrganizerRoutes(g *echo.Group) {
	g.POST("/sheet/:sheetID/organizer", func(c echo.Context) error {
		ctx := context.Background()
		sheetID, err := strconv.Atoi(c.Param("sheetID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Sheet ID is not a number: %s", c.Param("sheetID"))).SetInternal(err)
		}

		principalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetOrganizer, err := s.store.FindSheetOrganizer(ctx, &api.SheetOrganizerFind{
			SheetID:     &sheetID,
			PrincipalID: &principalID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sheet organizer with sheet ID %d and principal ID %d", sheetID, principalID)).SetInternal(err)
		}

		if sheetOrganizer == nil {
			sheetOrganizerCreate := &api.SheetOrganizerCreate{
				SheetID:     sheetID,
				PrincipalID: principalID,
			}
			if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetOrganizerCreate); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformatted post sheet organizer request").SetInternal(err)
			}

			if _, err := s.store.CreateSheetOrganizer(ctx, sheetOrganizerCreate); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create organizer %d to sheet %d", principalID, sheetID)).SetInternal(err)
			}
		} else {
			sheetOrganizerPatch := &api.SheetOrganizerPatch{
				ID:          sheetOrganizer.ID,
				SheetID:     sheetID,
				PrincipalID: principalID,
			}
			if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetOrganizerPatch); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformatted post sheet organizer request").SetInternal(err)
			}

			if _, err := s.store.PatchSheetOrganizer(ctx, sheetOrganizerPatch); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update sheet organizer with sheet ID %d and principal ID %d", sheetID, principalID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}
