package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/segment"
	"github.com/bytebase/bytebase/store"

	// Register all metric collectors.
	_ "github.com/bytebase/bytebase/store/metric-colloctor/instance-count"
	_ "github.com/bytebase/bytebase/store/metric-colloctor/issue-count"
	_ "github.com/bytebase/bytebase/store/metric-colloctor/policy-count"
	_ "github.com/bytebase/bytebase/store/metric-colloctor/project-count"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(1) * time.Hour
	// identifyTraitForPlan is the trait key for subscription plan.
	identifyTraitForPlan = "plan"
	// identifyTraitForVersion is the trait key for bytebase version.
	identifyTraitForVersion = "version"
)

// MetricReporter is the metric reporter.
type MetricReporter struct {
	l *zap.Logger

	// subscription is the pointer to the server.subscription.
	// the subscription can be updated by users so we need the pointer to get the latest value.
	subscription *enterpriseAPI.Subscription
	// Version is the bytebase's version
	version     string
	workspaceID string
	reporter    metric.Reporter
	store       *store.Store
}

// NewMetricReporter creates a new metric scheduler.
func NewMetricReporter(logger *zap.Logger, server *Server, workspaceID string) *MetricReporter {
	r := segment.NewReporter(logger, server.profile.MetricConnectionKey, workspaceID)

	return &MetricReporter{
		l:            logger,
		subscription: &server.subscription,
		version:      server.profile.Version,
		workspaceID:  workspaceID,
		reporter:     r,
		store:        server.store,
	}
}

// Run will run the metric reporter.
func (m *MetricReporter) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(metricSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()

	m.l.Debug(fmt.Sprintf("Metrics reporter started and will run every %v", metricSchedulerInterval))

	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						m.l.Error("Metrics reporter PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				// identify will be triggered in every schedule loop so that we can update the latest workspace profile like subscription plan.
				m.identify()

				ctx := context.Background()
				for name, collector := range store.Collectors() {
					m.l.Debug("Run metric collector", zap.String("collector", string(name)))

					metricList, err := collector.Collect(ctx, m.l, m.store)
					if err != nil {
						m.l.Error(
							"Failed to collect metric",
							zap.String("collector", string(name)),
							zap.Error(err),
						)
						continue
					}

					for _, metric := range metricList {
						if err := m.reporter.Report(metric); err != nil {
							m.l.Error(
								"Failed to report metric",
								zap.String("metric", string(metric.Name())),
								zap.Error(err),
							)
						}
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

// Identify will identify the workspace and update the subscription plan.
func (m *MetricReporter) identify() {
	plan := api.FREE.String()
	if m.subscription != nil {
		plan = m.subscription.Plan.String()
	}
	if err := m.reporter.Identify(&metric.Identifier{
		ID: m.workspaceID,
		Labels: map[string]string{
			identifyTraitForPlan:    plan,
			identifyTraitForVersion: m.version,
		},
	}); err != nil {
		m.l.Debug("reporter identify failed", zap.Error(err))
	}
}
