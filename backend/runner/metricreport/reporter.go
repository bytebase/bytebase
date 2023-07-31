// Package metricreport is a runner reporting metrics.
package metricreport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric/segment"
	"github.com/bytebase/bytebase/backend/store"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(1) * time.Hour
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
	licenseService enterpriseAPI.LicenseService
	profile        *config.Profile
	reporter       metric.Reporter
	collectors     map[string]metric.Collector
	store          *store.Store
}

// NewReporter creates a new metric scheduler.
func NewReporter(store *store.Store, licenseService enterpriseAPI.LicenseService, profile *config.Profile, enabled bool) *Reporter {
	var r metric.Reporter
	if enabled {
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

	log.Debug(fmt.Sprintf("Metrics reporter started and will run every %v", metricSchedulerInterval))

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
						log.Error("Metrics reporter PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
					}
				}()

				ctx := context.Background()
				// identify will be triggered in every schedule loop so that we can update the latest workspace profile such as subscription plan.
				workspaceID, err := m.identify(ctx)
				if err != nil {
					log.Error("failed to report identifier", zap.Error(err))
					return
				}

				for name, collector := range m.collectors {
					log.Debug("Run metric collector", zap.String("collector", name))

					metricList, err := collector.Collect(ctx)
					if err != nil {
						log.Error(
							"Failed to collect metric",
							zap.String("collector", name),
							zap.Error(err),
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
		log.Error(
			"Failed to report metric",
			zap.String("metric", string(metric.Name)),
			zap.Error(err),
		)
	}
}

// Identify will identify the workspace and update the subscription plan.
func (m *Reporter) identify(ctx context.Context) (string, error) {
	workspaceID, err := m.store.GetWorkspaceID(ctx)
	if err != nil {
		return "", err
	}
	subscription := m.licenseService.LoadSubscription(ctx)
	plan := subscription.Plan.String()
	orgID := subscription.OrgID
	orgName := subscription.OrgName

	trial := "N"
	if subscription.Trialing {
		trial = "Y"
	}

	subscriptionStartDate := ""
	subscriptionEndDate := ""
	if subscription.Plan != api.FREE {
		subscriptionStartDate = time.Unix(subscription.StartedTs, 0).Format(time.RFC3339)
		subscriptionEndDate = time.Unix(subscription.ExpiresTs, 0).Format(time.RFC3339)
	}

	user, err := m.store.GetUserByID(ctx, api.PrincipalIDForFirstUser)
	if err != nil {
		log.Debug("unable to get the first principal user", zap.Int("id", api.PrincipalIDForFirstUser), zap.Error(err))
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
			identifyTraitForLastActiveTime:        time.Unix(m.profile.LastActiveTs, 0).String(),
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
		log.Error("failed to find the workspace id", zap.Error(err))
		return
	}
	m.reportMetric(workspaceID, metric)
}
