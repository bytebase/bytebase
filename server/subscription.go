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
		subscription := s.loadSubscription()

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

// feature will check if user has access to a specific feature.
func (s *Server) feature(feature api.FeatureType) bool {
	subscription := s.loadSubscription()
	return api.FeatureMatrix[feature][subscription.Plan]
}

// loadLicense will load current subscription by license.
// Return subscription with free plan if no license found.
func (server *Server) loadSubscription() *enterprise.Subscription {
	subscription := &enterpriseAPI.Subscription{
		Plan: api.TEAM,
		// -1 means not expire, just for free plan
		ExpiresTs:     -1,
		InstanceCount: 5,
	}
	license, _ := server.loadLicense()
	if license != nil {
		subscription = &enterpriseAPI.Subscription{
			Plan:          license.Plan,
			ExpiresTs:     license.ExpiresTs,
			InstanceCount: license.InstanceCount,
		}
	} else {
		// change instance quota for dev environment.
		if server.mode == "dev" {
			subscription.InstanceCount = 9999
		}
	}

	return subscription
}

// loadLicense will get and parse valid license from file.
func (server *Server) loadLicense() (*enterprise.License, error) {
	license, err := server.LicenseService.LoadLicense()
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			server.l.Debug("Failed to find license", zap.String("error", err.Error()))
		} else {
			server.l.Warn("Failed to load valid license", zap.String("error", err.Error()))
		}
		return nil, err
	}

	server.l.Info(
		"Load valid license",
		zap.String("plan", license.Plan.String()),
		zap.Int64("expiresTs", license.ExpiresTs),
		zap.Time("expiresAt", time.Unix(license.ExpiresTs, 0)),
		zap.Int("instanceCount", license.InstanceCount),
	)

	return license, nil
}
