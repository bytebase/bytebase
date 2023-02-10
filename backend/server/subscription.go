package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerSubscriptionRoutes(g *echo.Group) {
	g.GET("/subscription", func(c echo.Context) error {
		ctx := c.Request().Context()
		subscription := s.licenseService.LoadSubscription(ctx)
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &subscription); err != nil {
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

		// clear the trialing setting for dev test
		if patch.License == "" {
			fmt.Println("clear trialing")
			if err := s.store.DeleteSettingV2(ctx, api.SettingEnterpriseTrial); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete the trialing license").SetInternal(err)
			}
		}

		if err := s.licenseService.StoreLicense(ctx, patch); err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to store license").SetInternal(err)
		}

		subscription := s.licenseService.LoadSubscription(ctx)
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})

	g.POST("/subscription/trial", func(c echo.Context) error {
		ctx := c.Request().Context()

		create := &api.TrialPlanCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, create); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create trial request").SetInternal(err)
		}
		license := &enterpriseAPI.License{
			InstanceCount: create.InstanceCount,
			Seat:          create.Seat,
			ExpiresTs:     time.Now().AddDate(0, 0, create.Days).Unix(),
			IssuedTs:      time.Now().Unix(),
			Plan:          create.Type,
			// the subject format for license should be {org id in hub}.{subscription id in hub}
			// as we just need to simply generate the trialing license in console, we can use the workspace id instead.
			Subject:  fmt.Sprintf("%s.%s", s.workspaceID, ""),
			Trialing: true,
			OrgName:  s.workspaceID,
		}

		subscription := s.licenseService.LoadSubscription(ctx)
		basePlan := subscription.Plan
		if license.Plan.Priority() <= subscription.Plan.Priority() {
			// Ignore the request if the priority for the trial plan is lower than the current plan.
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
			if err := jsonapi.MarshalPayload(c.Response().Writer, &subscription); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
			}
			return nil
		}

		if subscription.Trialing {
			license.ExpiresTs = subscription.ExpiresTs
			license.IssuedTs = subscription.StartedTs
		}

		value, err := json.Marshal(license)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal license").SetInternal(err)
		}

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
			_, created, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
				Name:        api.SettingEnterpriseTrial,
				Value:       string(value),
				Description: "The trialing license.",
			}, principalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create license").SetInternal(err)
			}

			if created && subscription.Trialing {
				// For trial upgrade
				// Case 1: Users just have the SettingEnterpriseTrial, don't upload their license in SettingEnterpriseLicense.
				// Case 2: Users have the SettingEnterpriseLicense with team plan and trialing status.
				// In both cases, we can override the SettingEnterpriseLicense with an empty value to get the valid free trial.
				if err := s.licenseService.StoreLicense(ctx, &enterpriseAPI.SubscriptionPatch{
					UpdaterID: principalID,
					License:   "",
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

		s.licenseService.RefreshCache(ctx)
		subscription = s.licenseService.LoadSubscription(ctx)
		currentPlan := subscription.Plan
		if s.MetricReporter != nil {
			s.MetricReporter.Report(&metric.Metric{
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
		if err := jsonapi.MarshalPayload(c.Response().Writer, &subscription); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal subscription response").SetInternal(err)
		}
		return nil
	})
}
