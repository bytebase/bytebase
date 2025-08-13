// Package metricreport is a runner reporting metrics.
package metricreport

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric/segment"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	metricSchedulerInterval = 6 * time.Hour
	// identifyTraitForPlan is the trait key for subscription plan.
	identifyTraitForPlan = "plan"
	// identifyTraitForTrial is the trait key for trialing.
	identifyTraitForTrial = "trial"
	// identifyTraitForSubscriptionStartDate is the trait key for subscription start date.
	identifyTraitForSubscriptionStartDate = "subscription_start"
	// identifyTraitForSubscriptionEndDate is the trait key for subscription end date.
	identifyTraitForSubscriptionEndDate = "subscription_end"
	// identifyTraitForOrgID is the trait key for organization id.
	identifyTraitForOrgID = "org_id"
	// identifyTraitForOrgName is the trait key for organization name.
	identifyTraitForOrgName = "org_name"
	// identifyTraitForMode is the trait key for Bytebase service mode.
	identifyTraitForMode = "mode"
	// identifyTraitForLastActiveTime is the trait key for Bytebase last active time.
	identifyTraitForLastActiveTime = "last_active"
	// identifyTraitForVersion is the trait key for Bytebase version.
	identifyTraitForVersion = "version"
	// bytebaseServiceModeSaaS is the mode for Bytebase SaaS.
	bytebaseServiceModeSaaS = "saas"
	// bytebaseServiceModeSelfhost is the mode for Bytebase self-host.
	bytebaseServiceModeSelfhost = "self-host"
)

// Reporter is the metric reporter.
type Reporter struct {
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	reporter       metric.Reporter
	collectors     map[string]metric.Collector
	store          *store.Store
}

// NewReporter creates a new metric scheduler.
func NewReporter(store *store.Store, licenseService *enterprise.LicenseService, profile *config.Profile) *Reporter {
	var r metric.Reporter
	if profile.MetricConnectionKey != "" {
		r = segment.NewReporter(profile.MetricConnectionKey)
	} else {
		r = segment.NewMockReporter()
	}

	return &Reporter{
		licenseService: licenseService,
		profile:        profile,
		reporter:       r,
		collectors:     make(map[string]metric.Collector),
		store:          store,
	}
}

// Run will run the metric reporter.
func (m *Reporter) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(metricSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()

	slog.Debug(fmt.Sprintf("Metrics reporter started and will run every %v", metricSchedulerInterval))

	for {
		select {
		case <-ticker.C:
			go func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = errors.Errorf("%v", r)
						}
						slog.Error("Metrics reporter PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
					}
				}()

				ctx := context.Background()

				// Check if metric collection is enabled in workspace profile setting
				if !m.isMetricCollectionEnabled(ctx) {
					return
				}

				// identify will be triggered in every schedule loop so that we can update the latest workspace profile such as subscription plan.
				workspaceID, err := m.identify(ctx)
				if err != nil {
					slog.Error("failed to report identifier", log.BBError(err))
					return
				}

				for name, collector := range m.collectors {
					slog.Debug("Run metric collector", slog.String("collector", name))

					metricList, err := collector.Collect(ctx)
					if err != nil {
						slog.Error(
							"Failed to collect metric",
							slog.String("collector", name),
							log.BBError(err),
						)
						continue
					}

					for _, metric := range metricList {
						m.reportMetric(workspaceID, metric)
					}
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Close will close the metric reporter.
func (m *Reporter) Close() {
	m.reporter.Close()
}

// Register will register a metric collector.
func (m *Reporter) Register(metricName metric.Name, collector metric.Collector) {
	m.collectors[string(metricName)] = collector
}

func (m *Reporter) reportMetric(id string, metric *metric.Metric) {
	if err := m.reporter.Report(id, metric); err != nil {
		slog.Error(
			"Failed to report metric",
			slog.String("metric", string(metric.Name)),
			log.BBError(err),
		)
	}
}

// isMetricCollectionEnabled checks if metric collection is enabled in workspace profile setting.
func (m *Reporter) isMetricCollectionEnabled(ctx context.Context) bool {
	// Always disable metrics in demo mode
	if m.profile.Demo {
		return false
	}

	// No connection key means no metrics
	if m.profile.MetricConnectionKey == "" {
		return false
	}

	// Get workspace profile setting
	setting, err := m.store.GetSettingV2(ctx, storepb.SettingName_WORKSPACE_PROFILE)
	if err != nil {
		// If we can't get the setting, default to enabled for backward compatibility
		slog.Debug("Failed to get workspace profile setting for metric collection", log.BBError(err))
		return true
	}

	if setting == nil || setting.Value == "" {
		// Default to enabled if setting is not found
		return true
	}

	var workspaceProfile storepb.WorkspaceProfileSetting
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), &workspaceProfile); err != nil {
		slog.Debug("Failed to unmarshal workspace profile setting", log.BBError(err))
		// Default to enabled if we can't parse the setting
		return true
	}

	return workspaceProfile.EnableMetricCollection
}

// Identify will identify the workspace and update the subscription plan.
func (m *Reporter) identify(ctx context.Context) (string, error) {
	workspaceID, err := m.store.GetWorkspaceID(ctx)
	if err != nil {
		return "", err
	}
	subscription := m.licenseService.LoadSubscription(ctx)
	plan := subscription.Plan.String()
	orgID := ""
	orgName := ""

	trial := "N"
	if subscription.Trialing {
		trial = "Y"
	}

	subscriptionStartDate := ""
	subscriptionEndDate := ""
	if subscription.Plan != v1pb.PlanType_FREE && subscription.ExpiresTime != nil {
		subscriptionEndDate = subscription.ExpiresTime.AsTime().Format(time.RFC3339)
	}

	user, err := m.store.GetUserByID(ctx, common.PrincipalIDForFirstUser)
	if err != nil {
		slog.Debug("unable to get the first principal user", slog.Int("id", common.PrincipalIDForFirstUser), log.BBError(err))
	}
	email := ""
	name := ""
	if user != nil {
		email = user.Email
		name = user.Name
	}

	mode := bytebaseServiceModeSelfhost
	if m.profile.SaaS {
		mode = bytebaseServiceModeSaaS
	}

	if err := m.reporter.Identify(&metric.Identifier{
		ID:    workspaceID,
		Email: email,
		Name:  name,
		Labels: map[string]string{
			identifyTraitForPlan:                  plan,
			identifyTraitForTrial:                 trial,
			identifyTraitForVersion:               m.profile.Version,
			identifyTraitForOrgID:                 orgID,
			identifyTraitForOrgName:               orgName,
			identifyTraitForMode:                  mode,
			identifyTraitForLastActiveTime:        time.Unix(m.profile.LastActiveTS.Load(), 0).String(),
			identifyTraitForSubscriptionStartDate: subscriptionStartDate,
			identifyTraitForSubscriptionEndDate:   subscriptionEndDate,
		},
	}); err != nil {
		return workspaceID, err
	}

	return workspaceID, nil
}

// Report will report a metric.
func (m *Reporter) Report(ctx context.Context, metric *metric.Metric) {
	workspaceID, err := m.store.GetWorkspaceID(ctx)
	if err != nil {
		slog.Error("failed to find the workspace id", log.BBError(err))
		return
	}
	m.reportMetric(workspaceID, metric)
}
