package server

import (
	"net/http"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
)

func (s *Server) registerSubscriptionRoutes(g *echo.Group) {
	g.GET("/subscription", func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &s.subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/subscription", func(c echo.Context) error {
		patch := &enterpriseAPI.SubscriptionPatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, patch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create subscription request").SetInternal(err)
		}
		patch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if err := s.LicenseService.StoreLicense(patch); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to store license").SetInternal(err)
		}

		s.subscription = s.loadSubscription()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &s.subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})
}

// loadLicense will load current subscription by license.
// Return subscription with free plan if no license found.
func (s *Server) loadSubscription() enterpriseAPI.Subscription {
	subscription := enterpriseAPI.Subscription{
		Plan: api.FREE,
		// -1 means not expire, just for free plan
		ExpiresTs:     -1,
		InstanceCount: 5,
	}
	license, _ := s.loadLicense()
	if license != nil {
		subscription = enterpriseAPI.Subscription{
			Plan:          license.Plan,
			ExpiresTs:     license.ExpiresTs,
			StartedTs:     license.IssuedTs,
			InstanceCount: license.InstanceCount,
			Trialing:      license.Trialing,
			OrgID:         license.OrgID(),
		}
	}

	return subscription
}

// loadLicense will get and parse valid license from file.
func (s *Server) loadLicense() (*enterpriseAPI.License, error) {
	license, err := s.LicenseService.LoadLicense()
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			log.Debug("Failed to find license", zap.String("error", err.Error()))
		} else {
			log.Warn("Failed to load valid license", zap.String("error", err.Error()))
		}
		return nil, err
	}

	log.Info(
		"Load valid license",
		zap.String("plan", license.Plan.String()),
		zap.Time("expiresAt", time.Unix(license.ExpiresTs, 0)),
		zap.Int("instanceCount", license.InstanceCount),
	)

	return license, nil
}

func (s *Server) feature(feature api.FeatureType) bool {
	return api.FeatureMatrix[feature][s.getEffectivePlan()]
}

func (s *Server) getPlanLimitValue(name api.PlanLimit) int64 {
	v, ok := api.PlanLimitValues[name]
	if !ok {
		return 0
	}
	return v[s.getEffectivePlan()]
}

func (s *Server) getEffectivePlan() api.PlanType {
	if expireTime := time.Unix(s.subscription.ExpiresTs, 0); expireTime.Before(time.Now()) {
		return api.FREE
	}
	return s.subscription.Plan
}
