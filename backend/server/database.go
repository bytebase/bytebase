package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/edit"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerDatabaseRoutes(g *echo.Group) {
	// When query metadata is present, we will return the schema metadata. Otherwise, we will return the raw dump.
	g.GET("/database/:databaseID/schema", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database %q", database.DatabaseName)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		dbSchema, err := s.store.GetDBSchema(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get dbSchema for database %q", database.DatabaseName)).SetInternal(err)
		}
		if dbSchema == nil {
			// TODO(d): make SyncDatabaseSchema return the updated database schema.
			if err := s.SchemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database schema for database %q", database.DatabaseName)).SetInternal(err)
			}
			newDBSchema, err := s.store.GetDBSchema(ctx, id)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get dbSchema for database %q", database.DatabaseName)).SetInternal(err)
			}
			if newDBSchema == nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("New dbSchema not found for database %q", database.DatabaseName)).SetInternal(err)
			}
			dbSchema = newDBSchema
		}

		if c.QueryParam("sdl") == "true" {
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
			if err != nil {
				return err
			}
			// We only support MySQL now.
			var engineType parser.EngineType
			switch instance.Engine {
			case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
				engineType = parser.MySQL
			default:
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Not support SDL format for %s instance", instance.Engine))
			}

			sdlSchema, err := transform.SchemaTransform(engineType, string(dbSchema.Schema))
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to transform SDL format for database %q", database.DatabaseName)).SetInternal(err)
			}

			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
			if _, err := c.Response().Write([]byte(sdlSchema)); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write schema response for database %q", database.DatabaseName)).SetInternal(err)
			}
		} else {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
			if _, err := c.Response().Write([]byte(dbSchema.Schema)); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write schema response for database %q", database.DatabaseName)).SetInternal(err)
			}
		}
		return nil
	})

	g.POST("/database/:databaseID/edit", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get database").SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get instance").SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance not found for database ID %d", databaseID))
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
		}
		databaseEdit := &api.DatabaseEdit{}
		if err := json.Unmarshal(body, databaseEdit); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed post database edit request").SetInternal(err)
		}

		engineType := parser.EngineType(instance.Engine)
		validateResultList, err := edit.ValidateDatabaseEdit(engineType, databaseEdit)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate DatabaseEdit").SetInternal(err)
		}

		databaseEditResult := &api.DatabaseEditResult{
			Statement:          "",
			ValidateResultList: validateResultList,
		}
		if len(validateResultList) == 0 {
			statement, err := edit.DeparseDatabaseEdit(engineType, databaseEdit)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to deparse DatabaseEdit").SetInternal(err)
			}
			databaseEditResult.Statement = statement
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, databaseEditResult); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database edit result response").SetInternal(err)
		}
		return nil
	})
}
