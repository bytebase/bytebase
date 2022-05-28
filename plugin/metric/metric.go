package metric

// Name is the metric name.
type Name string

// Metric is the API message for metric.
type Metric struct {
	Name   Name
	Value  int
	Labels map[string]string
}

// Identity is the Identity for metric.
type Identity struct {
	ID     string
	Labels map[string]string
}
