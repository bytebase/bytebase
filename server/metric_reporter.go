package server

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	metricCollector "github.com/bytebase/bytebase/plugin/metric/collector"
	metricReporter "github.com/bytebase/bytebase/plugin/metric/reporter"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(24) * time.Hour
)

// MetricScheduler is the metric scheduler.
type MetricScheduler struct {
	l *zap.Logger

	server        *Server
	reporter      api.MetricReporter
	deploymentID  string
	collectorList []api.MetricCollector
}

// NewMetricScheduler creates a new metric scheduler.
func NewMetricScheduler(logger *zap.Logger, server *Server, deploymentID string) *MetricScheduler {
	reporter := metricReporter.NewSegmentReporter(logger, server.profile.SegmentKey, deploymentID)
	collectorList := []api.MetricCollector{
		metricCollector.NewInstanceCollector(logger, server.store),
		metricCollector.NewIssueCollector(logger, server.store),
	}

	return &MetricScheduler{
		l:             logger,
		server:        server,
		deploymentID:  deploymentID,
		reporter:      reporter,
		collectorList: collectorList,
	}
}

// Run will run the metric scheduler.
func (m *MetricScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	plan := api.FREE.String()
	if m.server.subscription != nil {
		plan = m.server.subscription.Plan.String()
	}
	if err := m.reporter.Identify(&api.Workspace{
		Plan:         plan,
		DeploymentID: m.deploymentID,
	}); err != nil {
		m.l.Debug("reporter identify failed", zap.Error(err))
	}

	ticker := time.NewTicker(metricSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	m.l.Debug(fmt.Sprintf("Metrics scheduler started and will run every %v", metricSchedulerInterval))

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
