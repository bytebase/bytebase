package server

import (
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerDebugRoutes(g *echo.Group) {
	g.GET("/debug", func(c echo.Context) error {
		return s.currentDebugState(c)
	})

	g.PATCH("/debug", func(c echo.Context) error {
		var debugPatch api.DebugPatch
		if err := jsonapi.UnmarshalPayload(c.Request().Body, &debugPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to unmarshal debug patch request").SetInternal(err)
		}

		lvl := zap.InfoLevel
		if debugPatch.IsDebug {
			lvl = zap.DebugLevel
		}
		log.SetLevel(lvl)

		s.e.Debug = debugPatch.IsDebug

		return s.currentDebugState(c)
	})
}

func (s *Server) currentDebugState(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	if err := jsonapi.MarshalPayload(c.Response().Writer, &api.Debug{IsDebug: log.EnabledLevel(zap.DebugLevel)}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal debug info response").SetInternal(err)
	}
	return nil
}
