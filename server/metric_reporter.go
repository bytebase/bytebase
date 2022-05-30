package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/segment"

	"go.uber.org/zap"
)

const (
	metricSchedulerInterval = time.Duration(1) * time.Hour
)

// MetricReporter is the metric reporter.
type MetricReporter struct {
	identifier metric.Identifier
	reporter   metric.Reporter
	collectors map[string]metric.Collector
}

// NewMetricReporter creates a new metric scheduler.
func NewMetricReporter(workspaceID string, connectionKey string, identifier metric.Identifier) *MetricReporter {
	r := segment.NewReporter(connectionKey, workspaceID)

	return &MetricReporter{
		identifier: identifier,
		reporter:   r,
		collectors: make(map[string]metric.Collector),
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
							err = fmt.Errorf("%v", r)
						}
						log.Error("Metrics reporter PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				ctx := context.Background()
				// identify will be triggered in every schedule loop so that we can update the latest workspace profile like subscription plan.
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
						if err := m.reporter.Report(metric); err != nil {
							log.Error(
								"Failed to report metric",
								zap.String("metric", string(metric.Name)),
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

// Register will register a metric collector.
func (m *MetricReporter) Register(metricName metric.Name, collector metric.Collector) {
	m.collectors[string(metricName)] = collector
}

// Identify will identify the workspace and update the subscription plan.
func (m *MetricReporter) identify(ctx context.Context) {
	identity, err := m.identifier.Identify(ctx)
	if err != nil {
		log.Debug("collect identity failed", zap.Error(err))
		return
	}
	if err := m.reporter.Identify(identity); err != nil {
		log.Debug("reporter identify failed", zap.Error(err))
	}
}
