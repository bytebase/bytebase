package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

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
		_, created, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
			Name:  api.SettingEnterpriseTrial,
			Value: string(value),
		}, api.SystemBotID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create license").SetInternal(err)
		}
		if !created {
			return echo.NewHTTPError(http.StatusBadRequest, "your trial already exists")
		}

		// we need to override the SettingEnterpriseLicense with an empty value to get the valid free trial.
		if err := s.licenseService.StoreLicense(ctx, &enterpriseAPI.SubscriptionPatch{
			UpdaterID: api.SystemBotID,
			License:   "",
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to patch license").SetInternal(err)
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
