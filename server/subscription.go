package server

import (
	"net/http"

	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerSubscriptionRoutes(g *echo.Group) {
	g.GET("/subscription", func(c echo.Context) error {
		subscription, err := s.loadSubscription()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to load subscription").SetInternal(err)
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

		subscription, err := s.loadSubscription()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to load subscription").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) loadSubscription() (*enterpriseAPI.Subscription, error) {
	license, err := s.LicenseService.LoadLicense()
	if err != nil {
		return nil, err
	}

	s.license = license
	subscription := &enterpriseAPI.Subscription{
		Plan:          license.Plan,
		ExpiresTs:     license.ExpiresTs,
		InstanceCount: license.InstanceCount,
	}

	return subscription, nil
}
