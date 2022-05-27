package collector

import (
	"context"

	"github.com/bytebase/bytebase/plugin/metric"
)

// MetricCollector is the API message for metric collector.
type MetricCollector interface {
	Collect(ctx context.Context) ([]*metric.Metric, error)
}

// MetricIdentifier is the API message for metric identifier.
type MetricIdentifier interface {
	Collect(ctx context.Context) (*metric.Identifier, error)
}
