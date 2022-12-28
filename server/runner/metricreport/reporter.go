// Package metricreport is a runner reporting metrics.
package metricreport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/segment"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/store"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(1) * time.Hour
	// identifyTraitForPlan is the trait key for subscription plan.
	identifyTraitForPlan = "plan"
	// identifyTraitForOrgID is the trait key for organization id.
	identifyTraitForOrgID = "org_id"
	// identifyTraitForOrgName is the trait key for organization name.
	identifyTraitForOrgName = "org_name"
	// identifyTraitForVersion is the trait key for Bytebase version.
	identifyTraitForVersion = "version"
)

// Reporter is the metric reporter.
type Reporter struct {
	licenseService enterpriseAPI.LicenseService
	// Version is the bytebase's version
	version     string
	workspaceID string
	reporter    metric.Reporter
	collectors  map[string]metric.Collector
	store       *store.Store
}

// NewReporter creates a new metric scheduler.
func NewReporter(store *store.Store, licenseService enterpriseAPI.LicenseService, profile config.Profile, workspaceID string) *Reporter {
	r := segment.NewReporter(profile.MetricConnectionKey, workspaceID)

	return &Reporter{
		licenseService: licenseService,
		version:        profile.Version,
		workspaceID:    workspaceID,
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
			func() {
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
				m.Identify(ctx)
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
						m.Report(metric)
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

// Identify will identify the workspace and update the subscription plan.
func (m *Reporter) Identify(ctx context.Context) {
	subscription := m.licenseService.LoadSubscription(ctx)
	plan := subscription.Plan.String()
	orgID := subscription.OrgID
	orgName := subscription.OrgName

	principal, err := m.store.GetPrincipalByID(ctx, api.PrincipalIDForFirstUser)
	if err != nil {
		log.Debug("unable to get the first principal user", zap.Int("id", api.PrincipalIDForFirstUser), zap.Error(err))
	}
	email := ""
	name := ""
	if principal != nil {
		email = principal.Email
		name = principal.Name
	}

	if err := m.reporter.Identify(&metric.Identifier{
		ID:    m.workspaceID,
		Email: email,
		Name:  name,
		Labels: map[string]string{
			identifyTraitForPlan:    plan,
			identifyTraitForVersion: m.version,
			identifyTraitForOrgID:   orgID,
			identifyTraitForOrgName: orgName,
		},
	}); err != nil {
		log.Debug("reporter identify failed", zap.Error(err))
	}
}

// Report will report a metric.
func (m *Reporter) Report(metric *metric.Metric) {
	if err := m.reporter.Report(metric); err != nil {
		log.Error(
			"Failed to report metric",
			zap.String("metric", string(metric.Name)),
			zap.Error(err),
		)
	}
}
