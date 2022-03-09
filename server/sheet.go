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

func (s *Server) registerSheetRoutes(g *echo.Group) {
	g.POST("/sheet", func(c echo.Context) error {
		ctx := context.Background()
		sheetCreate := &api.SheetCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create sheet request").SetInternal(err)
		}

		if sheetCreate.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sheet request, missing name")
		}

		// If sheetCreate.DatabaseID is not nil, use its associated ProjectID as the new sheet's ProjectID.
		if sheetCreate.DatabaseID != nil {
			database, err := s.DatabaseService.FindDatabase(ctx, &api.DatabaseFind{
				ID: sheetCreate.DatabaseID,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %d", *sheetCreate.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", *sheetCreate.DatabaseID))
			}

			sheetCreate.ProjectID = database.ProjectID
		}

		project, err := s.ProjectService.FindProject(ctx, &api.ProjectFind{
			ID: &sheetCreate.ProjectID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %d", sheetCreate.ProjectID)).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", sheetCreate.ProjectID))
		}

		sheetCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		sheet, err := s.SheetService.CreateSheet(ctx, sheetCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create sheet").SetInternal(err)
		}

		if err := s.composeSheetRelationship(ctx, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created sheet relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create sheet response").SetInternal(err)
		}
		return nil
	})

	g.GET("/sheet", func(c echo.Context) error {
		ctx := context.Background()
		sheetFind := &api.SheetFind{}
		creatorID := c.Get(getPrincipalIDContextKey()).(int)
		sheetFind.CreatorID = &creatorID

		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			sheetFind.RowStatus = &rowStatus
		}

		if projectIDStr := c.QueryParams().Get("projectId"); projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.QueryParam("projectId"))).SetInternal(err)
			}
			sheetFind.ProjectID = &projectID
		}

		if databaseIDStr := c.QueryParams().Get("databaseId"); databaseIDStr != "" {
			databaseID, err := strconv.Atoi(databaseIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.QueryParam("databaseId"))).SetInternal(err)
			}
			sheetFind.DatabaseID = &databaseID
		}

		if visibility := api.SheetVisibility(c.QueryParam("visibility")); visibility != "" {
			sheetFind.Visibility = &visibility
		}

		sheetList, err := s.SheetService.FindSheetList(ctx, sheetFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch sheet list").SetInternal(err)
		}

		for _, sheet := range sheetList {
			if err := s.composeSheetRelationship(ctx, sheet); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet relationship: %v", sheet.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sheet list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/sheet/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetFind := &api.SheetFind{
			ID: &id,
		}

		sheet, err := s.SheetService.FindSheet(ctx, sheetFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet ID: %v", id)).SetInternal(err)
		}
		if sheet == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
		}

		if err := s.composeSheetRelationship(ctx, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet relationship: %v", sheet.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal sheet response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/sheet/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetPatch := &api.SheetPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch sheet request").SetInternal(err)
		}

		sheet, err := s.SheetService.PatchSheet(ctx, sheetPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch sheet ID: %v", id)).SetInternal(err)
		}

		if err := s.composeSheetRelationship(ctx, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated sheet relationship: %v", sheet.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal patch sheet response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/sheet/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		sheetDelete := &api.SheetDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		err = s.SheetService.DeleteSheet(ctx, sheetDelete)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete sheet ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) composeSheetRelationship(ctx context.Context, sheet *api.Sheet) error {
	var err error

	sheet.Creator, err = s.composePrincipalByID(ctx, sheet.CreatorID)
	if err != nil {
		return err
	}

	sheet.Updater, err = s.composePrincipalByID(ctx, sheet.UpdaterID)
	if err != nil {
		return err
	}

	sheet.Project, err = s.composeProjectByID(ctx, sheet.ProjectID)
	if err != nil {
		return err
	}

	if sheet.DatabaseID != nil {
		sheet.Database, err = s.composeDatabaseByFind(ctx, &api.DatabaseFind{
			ID: sheet.DatabaseID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
