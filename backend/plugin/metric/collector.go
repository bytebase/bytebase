package metric

import (
	"context"
)

// Collector is the API message for metric collector.
type Collector interface {
	Collect(ctx context.Context) ([]*Metric, error)
}
