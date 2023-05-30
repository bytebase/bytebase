package server

import (
	"fmt"
	"net/http"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerSheetRoutes(g *echo.Group) {
	g.POST("/sheet", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetCreate := &api.SheetCreate{
			CreatorID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create sheet request").SetInternal(err)
		}
		sheetCreate.Type = api.SheetForSQL

		if sheetCreate.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sheet request, missing name")
		}
		if sheetCreate.Source == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sheet request, missing source")
		}

		// If sheetCreate.DatabaseID is not nil, use its associated ProjectID as the new sheet's ProjectID.
		if sheetCreate.DatabaseID != nil {
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: sheetCreate.DatabaseID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", *sheetCreate.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("database %d not found", *sheetCreate.DatabaseID)).SetInternal(err)
			}
			project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
			if err != nil {
				return err
			}
			sheetCreate.ProjectID = project.UID
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: &sheetCreate.ProjectID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %d", sheetCreate.ProjectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", sheetCreate.ProjectID))
		}
		projectPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return err
		}

		role := c.Get(getRoleContextKey()).(api.Role)
		if role != api.Owner && role != api.DBA {
			// Non-workspace Owner or DBA can only create sheet into the project where she has the membership.
			if !isProjectOwnerOrDeveloper(currentPrincipalID, projectPolicy) {
				return echo.NewHTTPError(http.StatusForbidden, "Must be a project owner or developer to create new sheet")
			}
		}

		sheet, err := s.store.CreateSheet(ctx, sheetCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create sheet").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create sheet response").SetInternal(err)
		}
		return nil
	})
}
