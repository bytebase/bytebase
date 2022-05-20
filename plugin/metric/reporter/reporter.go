package reporter

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/metric/collector"
)

// MetricReporter is the API message for metric reporter.
type MetricReporter interface {
	Close()
	Report(metric *collector.Metric) error
	Identify(workspace *api.Workspace) error
}
