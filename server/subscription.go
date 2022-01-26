package server

import (
	"net/http"

	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerSubscriptionRoutes(g *echo.Group) {
	g.GET("/subscription", func(c echo.Context) error {
		if s.license == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to load subscription")
		}
		subscription := &enterpriseAPI.Subscription{
			Plan:          s.license.Plan,
			ExpiresTs:     s.license.ExpiresTs,
			InstanceCount: s.license.InstanceCount,
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/subscription", func(c echo.Context) error {
		patch := &enterpriseAPI.SubscriptionPatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, patch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create subscription request").SetInternal(err)
		}

		if err := s.LicenseService.StoreLicense(patch.License); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create license").SetInternal(err)
		}

		// Refresh license in memory
		s.LoadLicense(s.LicenseService)
		if s.license == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to load subscription")
		}
		subscription := &enterpriseAPI.Subscription{
			Plan:          s.license.Plan,
			ExpiresTs:     s.license.ExpiresTs,
			InstanceCount: s.license.InstanceCount,
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})
}
