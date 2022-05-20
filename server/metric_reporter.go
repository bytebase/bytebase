package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/metric"
	metricPlugin "github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/reporter"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(1) * time.Hour
	// identifyTraitForPlan is the trait key for subscription plan.
	identifyTraitForPlan = "plan"
)

// MetricReporter is the metric reporter.
type MetricReporter struct {
	l *zap.Logger

	// subscription is the pointer to the server.subscription.
	// the subscription can be updated by users so we need the pointer to get the latest value.
	subscription *enterpriseAPI.Subscription
	workspaceID  string
	reporter     *metricPlugin.Metric
}

// NewMetricReporter creates a new metric scheduler.
func NewMetricReporter(logger *zap.Logger, server *Server, workspaceID string) *MetricReporter {
	r := metricPlugin.NewMetric(logger, server.profile.MetricConnectionKey, workspaceID)

	r.Register(api.InstanceCountMetricName, metric.NewInstanceCollector(logger, server.store))
	r.Register(api.IssueCountMetricName, metric.NewIssueCollector(logger, server.store))

	return &MetricReporter{
		l:            logger,
		subscription: &server.subscription,
		workspaceID:  workspaceID,
		reporter:     r,
	}
}

// Run will run the metric reporter.
func (m *MetricReporter) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(metricSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()

	m.l.Debug(fmt.Sprintf("Metrics reporter started and will run every %v", metricSchedulerInterval))

	for {
		// identify will be triggered in every schedule loop so that we can update the latest workspace profile like subscription plan.
		m.identify()

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

				ctx := context.Background()
				m.reporter.CollectAndReport(ctx)
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
	if err := m.reporter.Identify(&reporter.MetricIdentifier{
		ID: m.workspaceID,
		Labels: map[string]string{
			identifyTraitForPlan: plan,
		},
	}); err != nil {
		m.l.Debug("reporter identify failed", zap.Error(err))
	}
}
