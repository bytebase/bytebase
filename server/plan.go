package server

import (
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerPlanRoutes(g *echo.Group) {
	g.GET("/plan", func(c echo.Context) error {
		plan := &api.Plan{
			Type: s.plan,
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, plan); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal plan response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/plan", func(c echo.Context) error {
		planPatch := &api.PlanPatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, planPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update plan request").SetInternal(err)
		}

		s.plan = planPatch.Type

		plan := &api.Plan{
			Type: s.plan,
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, plan); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal plan response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) feature(feature api.FeatureType) bool {
	plan := api.TEAM
	if s.license != nil {
		plan = s.license.Plan
	}

	return api.FeatureMatrix[feature][plan]
}
