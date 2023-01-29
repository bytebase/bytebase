package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

func (s *Server) registerPlanRoutes(g *echo.Group) {
	g.GET("/feature", func(c echo.Context) error {
		return c.JSON(http.StatusOK, api.FeatureMatrix)
	})
}
