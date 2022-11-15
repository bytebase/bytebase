package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) registerOpenAPIRoutesForInstance(g *echo.Group) {
	g.GET("/instance", s.listInstance)
}

func (s *Server) listInstance(c echo.Context) error {
	ctx := c.Request().Context()
	instanceFind := &api.InstanceFind{}
	if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
		rowStatus := api.RowStatus(rowStatusStr)
		instanceFind.RowStatus = &rowStatus
	}
	instanceList, err := s.store.FindInstance(ctx, instanceFind)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch instance list").SetInternal(err)
	}

	return c.JSON(http.StatusOK, instanceList)
}
