// Package metric is the interfaces for telemetry metrics.
package metric

// Name is the metric name.
type Name string

// Metric is the API message for metric.
type Metric struct {
	Name   Name
	Value  int
	Labels map[string]interface{}
}

// Identifier is the identifier for metric.
type Identifier struct {
	ID     string
	Email  string
	Name   string
	Labels map[string]string
}
