package collector

import "context"

// MetricName is the metric name.
type MetricName string

var (
	// instanceMetricName is the MetricName for instance count
	instanceCountMetricName MetricName = "bb.instance.count"
	// issueCountMetricName is the MetricName for issue count
	issueCountMetricName MetricName = "bb.issue.count"
)

// Metric is the API message for metric
type Metric struct {
	Name  MetricName
	Value int
	Label map[string]string
}

// MetricCollector is the API message for metric collector.
type MetricCollector interface {
	Collect(ctx context.Context) ([]*Metric, error)
}
