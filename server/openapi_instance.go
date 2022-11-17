package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	openAPIV1 "github.com/bytebase/bytebase/api/v1"
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

	var response []*openAPIV1.Instance
	for _, instance := range instanceList {
		response = append(response, &openAPIV1.Instance{
			ID:              instance.ID,
			RowStatus:       instance.RowStatus,
			CreatorID:       instance.CreatorID,
			UpdaterID:       instance.UpdaterID,
			CreatedTs:       instance.CreatedTs,
			UpdatedTs:       instance.UpdatedTs,
			EnvironmentName: instance.Environment.Name,
			Name:            instance.Name,
			Engine:          instance.Engine,
			EngineVersion:   instance.EngineVersion,
			ExternalLink:    instance.ExternalLink,
			Host:            instance.Host,
			Port:            instance.Port,
			Username:        instance.Username,
		})
	}

	return c.JSON(http.StatusOK, response)
}
