package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// instanceCollector is the metric data collector for instance.
type instanceCollector struct {
	l     *zap.Logger
	store *store.Store
}

// instanceEventName is the MetricEventName for instance
var instanceEventName MetricEventName = "bb.instance"

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

	status := api.Normal
	instanceList, err := c.store.FindInstance(ctx, &api.InstanceFind{
		RowStatus: &status,
	})
	if err != nil {
		return nil, err
	}

	instanceEngineMap := make(map[db.Type]int)
	for _, instance := range instanceList {
		instanceEngineMap[instance.Engine] = instanceEngineMap[instance.Engine] + 1
	}

	for engine, count := range instanceEngineMap {
		res = append(res, &Metric{
			EventName: instanceEventName,
			Properties: map[string]interface{}{
				"database": string(engine),
				"count":    count,
			},
		})
	}

	return res, nil
}
