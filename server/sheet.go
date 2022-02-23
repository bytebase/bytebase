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

		_, err := s.composeProjectByID(ctx, sheetCreate.ProjectID)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", sheetCreate.ProjectID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch project ID: %v", sheetCreate.ProjectID)).SetInternal(err)
		}

		_, err = s.composeInstanceByID(ctx, sheetCreate.InstanceID)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", sheetCreate.InstanceID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", sheetCreate.InstanceID)).SetInternal(err)
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

		if instanceIDStr := c.QueryParams().Get("instanceId"); instanceIDStr != "" {
			instanceID, err := strconv.Atoi(instanceIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Instance ID is not a number: %s", c.QueryParam("instanceId"))).SetInternal(err)
			}
			sheetFind.InstanceID = &instanceID
		}

		if databaseIDStr := c.QueryParams().Get("databaseId"); databaseIDStr != "" {
			databaseID, err := strconv.Atoi(databaseIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.QueryParam("databaseId"))).SetInternal(err)
			}
			sheetFind.DatabaseID = &databaseID
		}

		if projectIDStr := c.QueryParams().Get("projectId"); projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.QueryParam("projectId"))).SetInternal(err)
			}
			sheetFind.ProjectID = &projectID
		}

		if visibility := api.SheetVisibility(c.QueryParam("visibiliy")); visibility != "" {
			sheetFind.Visibility = &visibility
		}

		list, err := s.SheetService.FindSheetList(ctx, sheetFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch sheet list").SetInternal(err)
		}

		for _, sheet := range list {
			if err := s.composeSheetRelationship(ctx, sheet); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet relationship: %v", sheet.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
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

	sheet.Instance, err = s.composeInstanceByID(ctx, sheet.InstanceID)
	if err != nil {
		return err
	}

	if sheet.DatabaseID != nil {
		sheet.Database, err = s.composeDatabaseByID(ctx, *sheet.DatabaseID)
		if err != nil {
			return err
		}
	}

	return nil
}
