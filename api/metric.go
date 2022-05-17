package api

import (
	"context"
)

// Workspace is the instance for console application.
type Workspace struct {
	Plan string
	ID   string
}

// MetricEventName is the segment track event name.
type MetricEventName string

// Metric is the API message for metric
type Metric struct {
	EventName  MetricEventName
	Properties map[string]interface{}
}

// MetricReporter is the API message for metric reporter.
type MetricReporter interface {
	Close()
	Report(metric *Metric) error
	Identify(workspace *Workspace) error
}

// MetricCollector is the API message for metric collector.
type MetricCollector interface {
	Collect(ctx context.Context) ([]*Metric, error)
}
