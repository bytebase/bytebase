package collector

import "context"

// MetricName is the metric name.
type MetricName string

var (
	// instanceMetricName is the MetricName for instance count
	instanceCountMetricName MetricName = "bb.instance.count"
	// issueCountMetricName is the MetricName for issue count
	issueCountMetricName MetricName = "bb.issue.count"
	// projectCountMetricName is the MetricName for project count
	projectCountMetricName MetricName = "bb.project.count"
	// policyCountMetricName is the MetricName for policy count
	policyCountMetricName MetricName = "bb.policy.count"
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
