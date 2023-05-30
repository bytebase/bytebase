package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerAnomalyRoutes(g *echo.Group) {
	g.GET("/anomaly", func(c echo.Context) error {
		ctx := c.Request().Context()
		normalRowStatus := api.Normal
		find := &store.ListAnomalyMessage{
			RowStatus: &normalRowStatus,
		}

		if instanceIDStr := c.QueryParam("instance"); instanceIDStr != "" {
			instanceID, err := strconv.Atoi(instanceIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter instance is not a number: %s", instanceIDStr)).SetInternal(err)
			}
			find.InstanceUID = &instanceID
			find.Types = append(find.Types, api.AnomalyInstanceConnection, api.AnomalyInstanceMigrationSchema)
		}
		if databaseIDStr := c.QueryParam("database"); databaseIDStr != "" {
			databaseID, err := strconv.Atoi(databaseIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter database is not a number: %s", databaseIDStr)).SetInternal(err)
			}
			find.DatabaseUID = &databaseID
		}
		if rowStatus := c.QueryParam("rowStatus"); rowStatus != "" {
			find.RowStatus = (*api.RowStatus)(&rowStatus)
		}
		if anomalyType := c.QueryParam("type"); anomalyType != "" {
			find.Types = append(find.Types, api.AnomalyType(anomalyType))
		}

		anomalyList, err := s.store.ListAnomalyV2(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch anomaly list").SetInternal(err)
		}

		var apiAnomalyList []*api.Anomaly
		for _, anomaly := range anomalyList {
			apiAnomalyList = append(apiAnomalyList, anomaly.ToAPIAnomaly())
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, apiAnomalyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal anomaly list response").SetInternal(err)
		}
		return nil
	})
}
