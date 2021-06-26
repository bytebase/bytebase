package server

import (
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerActuatorRoutes(g *echo.Group) {
	g.GET("/actuator/info", func(c echo.Context) error {
		serverInfo := api.ServerInfo{
			Mode:      s.mode,
			Host:      s.host,
			Port:      strconv.Itoa(s.port),
			StartedTs: s.startedTs,
		}
		return c.JSON(http.StatusOK, serverInfo)
	})
}
