package api

import (
	"context"

	"github.com/bytebase/bytebase/plugin/db"
)

// MetricName is the metric name.
type MetricName string

var (
	// InstanceCountMetricName is the MetricName for instance count
	InstanceCountMetricName MetricName = "bb.instance.count"
	// IssueCountMetricName is the MetricName for issue count
	IssueCountMetricName MetricName = "bb.issue.count"
)

// Metric is the API message for metric
type Metric struct {
	Name   MetricName
	Value  int
	Labels map[string]string
}

// MetricCollector is the API message for metric collector.
type MetricCollector interface {
	Collect(ctx context.Context) ([]*Metric, error)
}

// MetricReporter is the API message for metric reporter.
type MetricReporter interface {
	Close()
	Report(metric *Metric) error
	Identify(workspace *Workspace) error
}

// InstanceCountMetric is the API message for instance count metric
type InstanceCountMetric struct {
	Engine        db.Type
	EnvironmentID int
	Count         int
}

// IssueCountMetric is the API message for issue count metric
type IssueCountMetric struct {
	Type  IssueType
	Count int
}
