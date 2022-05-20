package metric

import (
	"context"

	"github.com/bytebase/bytebase/plugin/metric/collector"
	"github.com/bytebase/bytebase/plugin/metric/reporter"
	"go.uber.org/zap"
)

// Metric is the metric plugin.
type Metric struct {
	l         *zap.Logger
	reporter  reporter.MetricReporter
	executors map[string]collector.MetricCollector
}

// NewMetric returns a new instance of Controller
func NewMetric(l *zap.Logger, key string, identifier string) *Metric {
	return &Metric{
		l:         l,
		reporter:  reporter.NewSegmentReporter(l, key, identifier),
		executors: make(map[string]collector.MetricCollector),
	}
}

// Close will close the reporter client.
func (m *Metric) Close() {
	m.reporter.Close()
}

// Register will register a metric collector.
func (m *Metric) Register(metricName reporter.MetricName, collector collector.MetricCollector) {
	m.executors[string(metricName)] = collector
}

// CollectAndReport will exec all collectors and report metric.
func (m *Metric) CollectAndReport(ctx context.Context) {
	for name, collector := range m.executors {
		m.l.Debug("Run metric collector", zap.String("collector", name))

		metricList, err := collector.Collect(ctx)
		if err != nil {
			m.l.Error(
				"Failed to collect metric",
				zap.String("collector", name),
				zap.Error(err),
			)
			continue
		}

		for _, metric := range metricList {
			if err := m.reporter.Report(metric); err != nil {
				m.l.Error(
					"Failed to report metric",
					zap.String("metric", string(metric.Name)),
					zap.Error(err),
				)
			}
		}
	}
}

// Identify will identify the metric with id.
func (m *Metric) Identify(identifier *reporter.MetricIdentifier) error {
	return m.reporter.Identify(identifier)
}
