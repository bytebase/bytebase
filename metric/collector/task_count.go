package collector

import (
	"context"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/collector"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// taskCollector is the metric data collector for task.
type taskCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewTaskCollector creates a new instance of taskCollector.
func NewTaskCollector(l *zap.Logger, store *store.Store) collector.MetricCollector {
	return &taskCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for task.
func (c *taskCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	taskCountMetricList, err := c.store.CountTaskGroupByTaskType(ctx)
	if err != nil {
		return nil, err
	}

	for _, taskCountMetric := range taskCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricAPI.TaskCountMetricName,
			Value: taskCountMetric.Count,
			Labels: map[string]string{
				"type": string(taskCountMetric.TaskType),
			},
		})
	}
	return res, nil
}
