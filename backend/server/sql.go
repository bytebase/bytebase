package server

import (
	"context"
	"net/http"

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
		if err = db.PingContext(context.Background()); err != nil {
			resultSet.Error = err.Error()
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})
}
