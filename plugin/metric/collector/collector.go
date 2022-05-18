package collector

import "context"

// MetricEventName is the segment track event name.
type MetricEventName string

// Metric is the API message for metric
type Metric struct {
	EventName  MetricEventName
	Properties map[string]interface{}
}

// MetricCollector is the API message for metric collector.
type MetricCollector interface {
	Collect(ctx context.Context) ([]*Metric, error)
}
