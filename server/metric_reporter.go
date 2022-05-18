package server

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/metric/collector"
	"github.com/bytebase/bytebase/plugin/metric/reporter"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(1) * time.Hour
)

// MetricScheduler is the metric scheduler.
type MetricScheduler struct {
	l *zap.Logger

	subscription  *enterpriseAPI.Subscription
	reporter      reporter.MetricReporter
	workspaceID   string
	collectorList []collector.MetricCollector
}

// NewMetricScheduler creates a new metric scheduler.
func NewMetricScheduler(logger *zap.Logger, server *Server, workspaceID string) *MetricScheduler {
	reporter := reporter.NewSegmentReporter(logger, server.profile.SegmentKey, workspaceID)
	collectorList := []collector.MetricCollector{
		collector.NewInstanceCollector(logger, server.store),
		collector.NewIssueCollector(logger, server.store),
	}

	return &MetricScheduler{
		l:             logger,
		subscription:  &server.subscription,
		workspaceID:   workspaceID,
		reporter:      reporter,
		collectorList: collectorList,
	}
}

// Run will run the metric scheduler.
func (m *MetricScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(metricSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	m.l.Debug(fmt.Sprintf("Metrics scheduler started and will run every %v", metricSchedulerInterval))

	for {
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
						m.l.Error("Metrics scheduler PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				ctx := context.Background()
				for _, collector := range m.collectorList {
					collectorName := reflect.TypeOf(collector).String()
					m.l.Debug("Run metric collector", zap.String("collector", collectorName))

					metricList, err := collector.Collect(ctx)
					if err != nil {
						m.l.Error(
							"Failed to collect metric",
							zap.String("collector", collectorName),
							zap.Error(err),
						)
						continue
					}
					for _, metric := range metricList {
						if err := m.reporter.Report(metric); err != nil {
							m.l.Error(
								"Failed to report metric",
								zap.String("metric", string(metric.EventName)),
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

// Close will close the metric client.
func (m *MetricScheduler) Close() {
	m.reporter.Close()
}

// Identify will identify the workspace and update the subscription plan.
func (m *MetricScheduler) identify() {
	plan := api.FREE.String()
	if m.subscription != nil {
		plan = m.subscription.Plan.String()
	}
	if err := m.reporter.Identify(&api.Workspace{
		Plan: plan,
		ID:   m.workspaceID,
	}); err != nil {
		m.l.Debug("reporter identify failed", zap.Error(err))
	}
}
