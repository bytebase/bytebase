package collector

import (
	"context"

	"github.com/bytebase/bytebase/plugin/metric/reporter"
)

// MetricCollector is the API message for metric collector.
type MetricCollector interface {
	Collect(ctx context.Context) ([]*reporter.Metric, error)
}
