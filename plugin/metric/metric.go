package metric

// Name is the metric name.
type Name string

// Metric is the API message for metric.
type Metric interface {
	Name() Name
	Value() int
	Labels() map[string]string
}

// Identifier is the identifier for metric.
type Identifier struct {
	ID     string
	Labels map[string]string
}
