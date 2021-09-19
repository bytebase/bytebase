package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerPolicyRoutes(g *echo.Group) {
	g.PATCH("/policy/:type", func(c echo.Context) error {
		policyUpsert := &api.PolicyUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, policyUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set policy request").SetInternal(err)
		}
		pType := api.PolicyType(c.Param("type"))
		policyUpsert.Type = pType
		policyUpsert.UpdaterId = c.Get(GetPrincipalIdContextKey()).(int)

		policy, err := s.PolicyService.UpsertPolicy(context.Background(), policyUpsert)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set policy for type %q", pType)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set policy response").SetInternal(err)
		}
		return nil
	})

	g.GET("/policy/:type", func(c echo.Context) error {
		policyFind := &api.PolicyFind{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, policyFind); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted get policy request").SetInternal(err)
		}
		pType := api.PolicyType(c.Param("type"))
		policyFind.Type = &pType

		policy, err := s.PolicyService.FindPolicy(context.Background(), policyFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get policy for type %q", pType)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, policy); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get policy response: %v", pType)).SetInternal(err)
		}
		return nil
	})
}
