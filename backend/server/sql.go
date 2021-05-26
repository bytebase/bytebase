package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/db"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerSqlRoutes(g *echo.Group) {
	g.POST("/sql/ping", func(c echo.Context) error {
		config := &api.SqlConfig{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, config); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql ping request").SetInternal(err)
		}

		db, err := db.Open(config.DBType, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: config.Username,
			Password: config.Password,
			Host:     config.Host,
			Port:     config.Port,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to open database").SetInternal(err)
		}

		resultSet := &api.SqlResultSet{}
		if err := db.Ping(context.Background()); err != nil {
			resultSet.Error = err.Error()
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})

	g.POST("/sql/syncschema", func(c echo.Context) error {
		sync := &api.SqlSyncSchema{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sync); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql sync schema request").SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), sync.InstanceId, true /*includeSecret*/)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", sync.InstanceId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", sync.InstanceId)).SetInternal(err)
		}

		db, err := db.Open(db.Mysql, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to open database").SetInternal(err)
		}

		resultSet := &api.SqlResultSet{}
		schemaList, err := db.SyncSchema(context.Background())
		if err != nil {
			resultSet.Error = err.Error()
		}

		workspaceId := api.DEFAULT_WORKPSACE_ID
		databaseFind := &api.DatabaseFind{
			WorkspaceId: &workspaceId,
			InstanceId:  &instance.ID,
		}
		for _, schema := range schemaList {
			databaseFind.Name = &schema.Name
			s.l.Infof("Schema11 %v", *schema)
			database, err := s.DatabaseService.FindDatabase(context.Background(), databaseFind)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					databaseCreate := &api.DatabaseCreate{
						CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
						WorkspaceId: api.DEFAULT_WORKPSACE_ID,
						ProjectId:   api.DEFAULT_PROJECT_ID,
						InstanceId:  instance.ID,
						Name:        schema.Name,
					}
					_, err := s.DatabaseService.CreateDatabase(context.Background(), databaseCreate)
					if err != nil {
						if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
							return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s. Failed to import new database: %s", instance.Name, databaseCreate.Name)).SetInternal(err)
					}
				} else {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s", instance.Name)).SetInternal(err)
				}
			} else {
				syncStatus := api.OK
				ts := time.Now().Unix()
				databasePatch := &api.DatabasePatch{
					ID:                   database.ID,
					WorkspaceId:          api.DEFAULT_WORKPSACE_ID,
					UpdaterId:            c.Get(GetPrincipalIdContextKey()).(int),
					SyncStatus:           &syncStatus,
					LastSuccessfulSyncTs: &ts,
				}
				_, err := s.DatabaseService.PatchDatabase(context.Background(), databasePatch)
				if err != nil {
					if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
						return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name))
					}
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s. Failed to update database: %s", instance.Name, database.Name)).SetInternal(err)
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})
}
