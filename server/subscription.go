package server

import (
	"net/http"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterprise "github.com/bytebase/bytebase/enterprise/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerSubscriptionRoutes(g *echo.Group) {
	g.GET("/subscription", func(c echo.Context) error {
		subscription := &enterpriseAPI.Subscription{
			Plan: api.FREE,
			// -1 means not expire, just for free plan
			ExpiresTs:     -1,
			InstanceCount: 5,
		}

		license, err := s.loadLicense()
		if err != nil {
			s.l.Error("Failed to get valid license for subscription", zap.String("error", err.Error()))
		} else {
			subscription = &enterpriseAPI.Subscription{
				Plan:          license.Plan,
				ExpiresTs:     license.ExpiresTs,
				InstanceCount: license.InstanceCount,
			}
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

		license, err := s.loadLicense()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to load subscription").SetInternal(err)
		}
		subscription := &enterpriseAPI.Subscription{
			Plan:          license.Plan,
			ExpiresTs:     license.ExpiresTs,
			InstanceCount: license.InstanceCount,
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})
}

// loadLicense will get and parse valid license from file.
func (server *Server) loadLicense() (*enterprise.License, error) {
	license, err := server.LicenseService.LoadLicense()
	if err != nil {
		if common.ErrorCode(err) == common.LicenseInvalid {
			server.l.Warn("Failed to load valid license", zap.String("error", err.Error()))
		} else {
			server.l.Debug("Failed to find license", zap.String("error", err.Error()))
		}
		return nil, err
	}

	server.l.Info(
		"Load valid license",
		zap.String("plan", license.Plan.String()),
		zap.Time("expiresAt", time.Unix(license.ExpiresTs, 0)),
		zap.Int("instanceCount", license.InstanceCount),
	)

	return license, nil
}
