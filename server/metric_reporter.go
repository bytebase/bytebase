package server

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
	"github.com/bytebase/bytebase/store"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(1) * time.Hour
	// identifyTraitForPlan is the trait key for subscription plan.
	identifyTraitForPlan = "plan"
	// identifyTraitForOrgID is the trait key for organization id.
	identifyTraitForOrgID = "org_id"
	// identifyTraitForVersion is the trait key for Bytebase version.
	identifyTraitForVersion = "version"
	// principalIDForFirstUser is the principal id for the first user in workspace.
	principalIDForFirstUser = 101
)

// MetricReporter is the metric reporter.
type MetricReporter struct {
	// subscription is the pointer to the server.subscription.
	// the subscription can be updated by users so we need the pointer to get the latest value.
	subscription *enterpriseAPI.Subscription
	// Version is the bytebase's version
	version     string
	workspaceID string
	reporter    metric.Reporter
	collectors  map[string]metric.Collector
	store       *store.Store
}

// NewMetricReporter creates a new metric scheduler.
func NewMetricReporter(server *Server, workspaceID string) *MetricReporter {
	r := segment.NewReporter(server.profile.MetricConnectionKey, workspaceID)

	return &MetricReporter{
		subscription: &server.subscription,
		version:      server.profile.Version,
		workspaceID:  workspaceID,
		reporter:     r,
		collectors:   make(map[string]metric.Collector),
		store:        server.store,
	}
}

// Run will run the metric reporter.
func (m *MetricReporter) Run(ctx context.Context, wg *sync.WaitGroup) {
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
				m.identify(ctx)
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
						m.report(metric)
					}
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Close will close the metric reporter.
func (m *MetricReporter) Close() {
	m.reporter.Close()
}

// Register will register a metric collector.
func (m *MetricReporter) Register(metricName metric.Name, collector metric.Collector) {
	m.collectors[string(metricName)] = collector
}

// Identify will identify the workspace and update the subscription plan.
func (m *MetricReporter) identify(ctx context.Context) {
	plan := api.FREE.String()
	orgID := ""
	orgName := ""

	if m.subscription != nil {
		plan = m.subscription.Plan.String()
		orgID = m.subscription.OrgID
		orgName = m.subscription.OrgName
	}

	principal, err := m.store.GetPrincipalByID(ctx, principalIDForFirstUser)
	if err != nil {
		log.Debug("unable to get the first principal user", zap.Int("id", principalIDForFirstUser), zap.Error(err))
	}
	email := ""
	if principal != nil {
		email = principal.Email
	}

	if err := m.reporter.Identify(&metric.Identifier{
		ID:    m.workspaceID,
		Email: email,
		Name:  orgName,
		Labels: map[string]string{
			identifyTraitForPlan:    plan,
			identifyTraitForVersion: m.version,
			identifyTraitForOrgID:   orgID,
		},
	}); err != nil {
		log.Debug("reporter identify failed", zap.Error(err))
	}
}

func (m *MetricReporter) report(metric *metric.Metric) {
	if err := m.reporter.Report(metric); err != nil {
		log.Error(
			"Failed to report metric",
			zap.String("metric", string(metric.Name)),
			zap.Error(err),
		)
	}
}
