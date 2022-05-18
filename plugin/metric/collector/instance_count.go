package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// instanceCollector is the metric data collector for instance.
type instanceCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewInstanceCollector creates a new instance of instanceCollector
func NewInstanceCollector(l *zap.Logger, store *store.Store) MetricCollector {
	return &instanceCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the netric for instance
func (c *instanceCollector) Collect(ctx context.Context) ([]*Metric, error) {
	var res []*Metric

	instanceCountMap, err := c.store.CountInstanceGroupByEngine(ctx, api.Normal)
	if err != nil {
		return nil, err
	}

	for engine, count := range instanceCountMap {
		res = append(res, &Metric{
			Name:  instanceCountMetricName,
			Value: count,
			Label: map[string]string{
				"database": string(engine),
			},
		})
	}

	return res, nil
}
