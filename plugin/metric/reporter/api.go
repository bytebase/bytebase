package reporter

// MetricName is the metric name.
type MetricName string

// Metric is the API message for metric.
type Metric struct {
	Name   MetricName
	Value  int
	Labels map[string]string
}

// MetricIdentifier is the identifier for metric.
type MetricIdentifier struct {
	ID     string
	Labels map[string]string
}

// MetricReporter is the API message for metric reporter.
type MetricReporter interface {
	Close()
	Report(metric *Metric) error
	Identify(identifier *MetricIdentifier) error
}
