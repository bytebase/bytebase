package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/db"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerMigrationRoutes(g *echo.Group) {
	g.POST("/migration/instance", func(c echo.Context) error {
		connectionInfo := &api.ConnectionInfo{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, connectionInfo); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted instance migration status request").SetInternal(err)
		}

		resultSet := &api.SqlResultSet{}
		db, err := db.Open(connectionInfo.DBType, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: connectionInfo.Username,
			Password: connectionInfo.Password,
			Host:     connectionInfo.Host,
			Port:     connectionInfo.Port,
		})
		if err != nil {
			resultSet.Error = err.Error()
		} else {
			if err := db.SetupMigrationIfNeeded(context.Background()); err != nil {
				resultSet.Error = err.Error()
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for host:port: %v:%v", connectionInfo.Host, connectionInfo.Port)).SetInternal(err)
		}
		return nil
	})

	g.POST("/migration/instance/status", func(c echo.Context) error {
		connectionInfo := &api.ConnectionInfo{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, connectionInfo); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted instance migration status request").SetInternal(err)
		}

		instanceMigration := &api.InstanceMigration{}
		db, err := db.Open(connectionInfo.DBType, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: connectionInfo.Username,
			Password: connectionInfo.Password,
			Host:     connectionInfo.Host,
			Port:     connectionInfo.Port,
		})
		if err != nil {
			instanceMigration.Status = api.InstanceMigrationUnknown
			instanceMigration.Error = err.Error()
		} else {
			setup, err := db.NeedsSetupMigration(context.Background())
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check migration setup status for host:port: %v:%v", connectionInfo.Host, connectionInfo.Port)).SetInternal(err)
			}
			if setup {
				instanceMigration.Status = api.InstanceMigrationNotExist
			} else {
				instanceMigration.Status = api.InstanceMigrationOK
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instanceMigration); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for host:port: %v:%v", connectionInfo.Host, connectionInfo.Port)).SetInternal(err)
		}
		return nil
	})
}
