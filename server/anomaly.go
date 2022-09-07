package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) registerAnomalyRoutes(g *echo.Group) {
	g.GET("/anomaly", func(c echo.Context) error {
		ctx := c.Request().Context()
		rowStatus := api.Normal
		anomalyFind := &api.AnomalyFind{
			RowStatus: &rowStatus,
		}
		if instanceIDStr := c.QueryParam("instance"); instanceIDStr != "" {
			instanceID, err := strconv.Atoi(instanceIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter instance is not a number: %s", instanceIDStr)).SetInternal(err)
			}
			anomalyFind.InstanceOnly = true
			anomalyFind.InstanceID = &instanceID
		}
		if databaseIDStr := c.QueryParam("database"); databaseIDStr != "" {
			databaseID, err := strconv.Atoi(databaseIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter database is not a number: %s", databaseIDStr)).SetInternal(err)
			}
			anomalyFind.DatabaseID = &databaseID
		}
		anomalyList, err := s.store.FindAnomaly(ctx, anomalyFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch anomaly list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, anomalyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal anomaly list response").SetInternal(err)
		}
		return nil
	})
}
