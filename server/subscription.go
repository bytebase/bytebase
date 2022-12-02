package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
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
		ctx := c.Request().Context()
		if err := jsonapi.UnmarshalPayload(c.Request().Body, patch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create subscription request").SetInternal(err)
		}
		patch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if err := s.LicenseService.StoreLicense(ctx, patch); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to store license").SetInternal(err)
		}

		s.subscription = s.loadSubscription(ctx)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &s.subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})

	g.POST("/subscription/trial", func(c echo.Context) error {
		create := &api.TrialPlanCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, create); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create trial request").SetInternal(err)
		}
		license := &enterpriseAPI.License{
			InstanceCount: create.InstanceCount,
			ExpiresTs:     time.Now().AddDate(0, 0, create.Days).Unix(),
			IssuedTs:      time.Now().Unix(),
			Plan:          create.Type,
			// the subject format for license should be {org id in hub}.{subscription id in hub}
			// as we just need to simply generate the trialing license in console, we can use the workspace id instead.
			Subject:  fmt.Sprintf("%s.%s", s.workspaceID, ""),
			Trialing: true,
			OrgName:  s.workspaceID,
		}

		upgradeTrial := s.subscription.Trialing && license.Plan.Priority() > s.subscription.Plan.Priority()
		if upgradeTrial {
			license.ExpiresTs = s.subscription.ExpiresTs
			license.IssuedTs = s.subscription.StartedTs
		}

		value, err := json.Marshal(license)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal license").SetInternal(err)
		}

		ctx := c.Request().Context()
		principalID := c.Get(getPrincipalIDContextKey()).(int)
		settingName := api.SettingEnterpriseTrial
		settings, err := s.store.FindSetting(ctx, &api.SettingFind{
			Name: &settingName,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find setting").SetInternal(err)
		}

		if len(settings) == 0 {
			// We will create a new setting named SettingEnterpriseTrial to store the free trial license.
			_, created, err := s.store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
				CreatorID:   principalID,
				Name:        api.SettingEnterpriseTrial,
				Value:       string(value),
				Description: "The trialing license.",
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create license").SetInternal(err)
			}

			if created && upgradeTrial {
				// For trial upgrade
				// Case 1: Users just have the SettingEnterpriseTrial, don't upload their license in SettingEnterpriseLicense.
				// Case 2: Users have the SettingEnterpriseLicense with team plan and trialing status.
				// In both cases, we can override the SettingEnterpriseLicense with an empty value to get the valid free trial.
				if _, err := s.store.PatchSetting(ctx, &api.SettingPatch{
					UpdaterID: principalID,
					Name:      api.SettingEnterpriseLicense,
					Value:     "",
				}); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to remove license").SetInternal(err)
				}
			}
		} else {
			// Update the existed free trial.
			if _, err := s.store.PatchSetting(ctx, &api.SettingPatch{
				UpdaterID: principalID,
				Name:      api.SettingEnterpriseTrial,
				Value:     string(value),
			}); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to patch license").SetInternal(err)
			}
		}

		basePlan := s.subscription.Plan
		s.subscription = s.loadSubscription(ctx)
		currentPlan := s.subscription.Plan

		if s.MetricReporter != nil {
			s.MetricReporter.report(&metric.Metric{
				Name:  metricAPI.SubscriptionTrialMetricName,
				Value: 1,
				Labels: map[string]interface{}{
					"trial_plan":    currentPlan.String(),
					"from_plan":     basePlan.String(),
					"lark_notified": false,
				},
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &s.subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})
}

// loadLicense will load current subscription by license.
// Return subscription with free plan if no license found.
func (s *Server) loadSubscription(ctx context.Context) enterpriseAPI.Subscription {
	subscription := enterpriseAPI.Subscription{
		Plan: api.FREE,
		// -1 means not expire, just for free plan
		ExpiresTs:     -1,
		InstanceCount: 5,
	}

	license, _ := s.LicenseService.LoadLicense(ctx)
	if license != nil {
		subscription = enterpriseAPI.Subscription{
			Plan:          license.Plan,
			ExpiresTs:     license.ExpiresTs,
			StartedTs:     license.IssuedTs,
			InstanceCount: license.InstanceCount,
			Trialing:      license.Trialing,
			OrgID:         license.OrgID(),
			OrgName:       license.OrgName,
		}
	}

	return subscription
}

func (s *Server) feature(feature api.FeatureType) bool {
	return api.Feature(feature, s.getEffectivePlan())
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
