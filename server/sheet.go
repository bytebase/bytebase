package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/store"
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

		if sheetCreate.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sheet request, missing name")
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
			if !hasActiveProjectMembership(currentPrincipalID, projectPolicy) {
				return echo.NewHTTPError(http.StatusUnauthorized, "Must be a project member to create new sheet")
			}
		}

		sheetCreate.Source = api.SheetFromBytebase
		sheetCreate.Type = api.SheetForSQL
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

	// Get current user created sheet list.
	g.GET("/sheet/my", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetFind := api.SheetFind{
			CreatorID: &currentPrincipalID,
		}

		if rowStatusStr := c.QueryParam("rowStatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			sheetFind.RowStatus = &rowStatus
		}

		mySheetList, err := s.store.FindSheet(ctx, &sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch sheet list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, mySheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal my sheet list response").SetInternal(err)
		}
		return nil
	})

	// Get sheet list which is shared with current user.
	// The desired sheets would be PROJECT sheets which current user is one of the active member of its project and all public sheets.
	g.GET("/sheet/shared", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetFind := api.SheetFind{
			PrincipalID: &currentPrincipalID,
		}

		var sheetList []*api.Sheet
		projectSheetVisibility := api.ProjectSheet
		sheetFind.Visibility = &projectSheetVisibility
		projectSheetList, err := s.store.FindSheet(ctx, &sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch shared project sheet list").SetInternal(err)
		}
		sheetList = append(sheetList, projectSheetList...)

		publicSheetVisibility := api.PublicSheet
		sheetFind.Visibility = &publicSheetVisibility
		publicSheetList, err := s.store.FindSheet(ctx, &sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch shared public sheet list").SetInternal(err)
		}
		sheetList = append(sheetList, publicSheetList...)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal shared sheet list response").SetInternal(err)
		}
		return nil
	})

	// Get current user starred sheet list.
	// The desired sheets would be any visibility but have a record of sheet organizer.
	g.GET("/sheet/starred", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		sheetFind := api.SheetFind{
			OrganizerPrincipalID: &currentPrincipalID,
		}

		starredSheetList, err := s.store.FindSheet(ctx, &sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch starred sheet list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, starredSheetList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal starred sheet list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/sheet/:sheetID", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		id, err := strconv.Atoi(c.Param("sheetID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("sheetID"))).SetInternal(err)
		}

		sheetFind := &api.SheetFind{
			ID: &id,
		}
		sheet, err := s.store.GetSheet(ctx, sheetFind, currentPrincipalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch sheet ID: %v", id)).SetInternal(err)
		}
		if sheet == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal sheet response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/sheet/:sheetID", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		id, err := strconv.Atoi(c.Param("sheetID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("sheetID"))).SetInternal(err)
		}

		sheetPatch := &api.SheetPatch{
			ID:        id,
			UpdaterID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sheetPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch sheet request").SetInternal(err)
		}

		sheet, err := s.store.PatchSheet(ctx, sheetPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sheet ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch sheet with ID: %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, sheet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal patch sheet response: %d", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/sheet/:sheetID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("sheetID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("sheetID"))).SetInternal(err)
		}

		sheetDelete := &api.SheetDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		err = s.store.DeleteSheet(ctx, sheetDelete)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete sheet ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

// SheetInfo represents the sheet related information from sheetPathTemplate.
type SheetInfo struct {
	EnvironmentID string
	DatabaseName  string
	SheetName     string
}

// parseSheetInfo matches sheetPath against sheetPathTemplate. If sheetPath matches, then it will derive SheetInfo from the sheetPath.
// Both sheetPath and sheetPathTemplate are the full file path(including the base directory) of the repository.
func parseSheetInfo(sheetPath string, sheetPathTemplate string) (*SheetInfo, error) {
	placeholderList := []string{
		"ENV_NAME",
		"DB_NAME",
		"NAME",
	}
	sheetPathRegex := sheetPathTemplate
	for _, placeholder := range placeholderList {
		sheetPathRegex = strings.ReplaceAll(sheetPathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9\\+\\-\\=\\_\\#\\!\\$\\. ]+)", placeholder))
	}
	sheetRegex, err := regexp.Compile(fmt.Sprintf("^%s$", sheetPathRegex))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid sheet path template %q", sheetPathTemplate)
	}
	if !sheetRegex.MatchString(sheetPath) {
		return nil, errors.Errorf("sheet path %q does not match sheet path template %q", sheetPath, sheetPathTemplate)
	}

	matchList := sheetRegex.FindStringSubmatch(sheetPath)
	sheetInfo := &SheetInfo{}
	for _, placeholder := range placeholderList {
		index := sheetRegex.SubexpIndex(placeholder)
		if index >= 0 {
			switch placeholder {
			case "ENV_NAME":
				sheetInfo.EnvironmentID = matchList[index]
			case "DB_NAME":
				sheetInfo.DatabaseName = matchList[index]
			case "NAME":
				sheetInfo.SheetName = matchList[index]
			}
		}
	}

	return sheetInfo, nil
}
