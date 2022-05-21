package reporter

import (
	"github.com/bytebase/bytebase/plugin/metric"
)

// MetricReporter is the API message for metric reporter.
type MetricReporter interface {
	Close()
	Report(metric *metric.Metric) error
	Identify(identifier *metric.Identifier) error
}
