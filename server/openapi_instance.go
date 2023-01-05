package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	openAPIV1 "github.com/bytebase/bytebase/api/v1"
	"github.com/bytebase/bytebase/store"
)

func (s *Server) updateInstanceDatabase(c echo.Context) error {
	ctx := c.Request().Context()
	instanceName := c.Param("instanceName")
	databaseName := c.Param("database")

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}
	instanceDatabasePatch := &openAPIV1.InstanceDatabasePatch{}
	if err := json.Unmarshal(body, instanceDatabasePatch); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch instance database request").SetInternal(err)
	}

	instances, err := s.store.FindInstance(ctx, &api.InstanceFind{Name: &instanceName})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find instance").SetInternal(err)
	}
	if len(instances) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Found %v instances with name %q but expecting one", len(instances), instanceName)
	}
	instance := instances[0]

	databases, err := s.store.FindDatabase(ctx, &api.DatabaseFind{InstanceID: &instance.ID, Name: &databaseName})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find database").SetInternal(err)
	}
	if len(databases) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Found %v databases with name %q but expecting one", len(databases), databaseName)
	}
	database := databases[0]

	var patchProjectID *int
	if instanceDatabasePatch.Project != nil {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: instanceDatabasePatch.Project})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find project").SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "project %q not found", *instanceDatabasePatch.Project)
		}
		patchProjectID = &project.UID
	}
	updaterID := c.Get(getPrincipalIDContextKey()).(int)
	if _, err := s.store.PatchDatabase(ctx, &api.DatabasePatch{ID: database.ID, UpdaterID: updaterID, ProjectID: patchProjectID}); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to patch database project").SetInternal(err)
	}
	return c.JSON(http.StatusOK, "")
}
