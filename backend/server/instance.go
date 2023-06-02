package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerInstanceRoutes(g *echo.Group) {
	g.GET("/instance/:instanceID/migration/history/:historyID", func(c echo.Context) error {
		ctx := c.Request().Context()
		instanceID, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Instance ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		historyID := c.Param("historyID")
		isSDL := c.QueryParam("sdl") == "true"

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", instanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", instanceID))
		}

		var entry *db.MigrationHistory
		find := &db.MigrationHistoryFind{ID: &historyID, InstanceID: &instanceID}
		list, err := s.store.FindInstanceChangeHistoryList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
		}
		if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Migration history ID %q not found for instance %q", historyID, instance.Title))
		}
		entry = list[0]

		if isSDL {
			var engineType parser.EngineType
			switch instance.Engine {
			case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
				engineType = parser.MySQL
			default:
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Not support SDL format for %s instance", instance.Engine))
			}
			entry.Schema, err = transform.SchemaTransform(engineType, entry.Schema)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to transform Schema to SDL format for instance %q", instance.Title)).SetInternal(err)
			}
			entry.SchemaPrev, err = transform.SchemaTransform(engineType, entry.SchemaPrev)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to transform SchemaPrev to SDL format for instance %q", instance.Title)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &api.MigrationHistory{
			ID:                    entry.ID,
			Creator:               entry.Creator,
			CreatedTs:             entry.CreatedTs,
			Updater:               entry.Updater,
			UpdatedTs:             entry.UpdatedTs,
			ReleaseVersion:        entry.ReleaseVersion,
			Database:              entry.Namespace,
			Source:                entry.Source,
			Type:                  entry.Type,
			Status:                entry.Status,
			Version:               entry.Version,
			UseSemanticVersion:    entry.UseSemanticVersion,
			SemanticVersionSuffix: entry.SemanticVersionSuffix,
			Description:           entry.Description,
			Statement:             entry.Statement,
			Schema:                entry.Schema,
			SchemaPrev:            entry.SchemaPrev,
			ExecutionDurationNs:   entry.ExecutionDurationNs,
			IssueID:               entry.IssueID,
			Payload:               entry.Payload,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Title)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/history", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
		}

		find := &db.MigrationHistoryFind{
			InstanceID: &instance.UID,
		}
		if databaseStr := c.QueryParams().Get("database"); databaseStr != "" {
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:   &instance.ResourceID,
				DatabaseName: &databaseStr,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get database %q from instance ID %v", databaseStr, instance.ResourceID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot find database %q from instance ID %v", databaseStr, instance.ResourceID)).SetInternal(err)
			}
			find.Database = &databaseStr
			find.DatabaseID = &database.UID
		}
		if versionStr := c.QueryParams().Get("version"); versionStr != "" {
			find.Version = &versionStr
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
			}
			find.Limit = &limit
		}

		migrationHistoryList, err := s.store.FindInstanceChangeHistoryList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
		}

		historyList := []*api.MigrationHistory{}
		for _, entry := range migrationHistoryList {
			historyList = append(historyList, &api.MigrationHistory{
				ID:                    entry.ID,
				Creator:               entry.Creator,
				CreatedTs:             entry.CreatedTs,
				Updater:               entry.Updater,
				UpdatedTs:             entry.UpdatedTs,
				ReleaseVersion:        entry.ReleaseVersion,
				Database:              entry.Namespace,
				Source:                entry.Source,
				Type:                  entry.Type,
				Status:                entry.Status,
				Version:               entry.Version,
				UseSemanticVersion:    entry.UseSemanticVersion,
				SemanticVersionSuffix: entry.SemanticVersionSuffix,
				Description:           entry.Description,
				Statement:             entry.Statement,
				Schema:                entry.Schema,
				SchemaPrev:            entry.SchemaPrev,
				ExecutionDurationNs:   entry.ExecutionDurationNs,
				IssueID:               entry.IssueID,
				Payload:               entry.Payload,
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, historyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Title)).SetInternal(err)
		}
		return nil
	})
}
