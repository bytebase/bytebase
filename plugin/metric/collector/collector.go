package collector

import (
	"context"

	"github.com/bytebase/bytebase/plugin/metric/reporter"
	"go.uber.org/zap"
)

// MetricCollectorController is the controller for metric collector
type MetricCollectorController struct {
	l         *zap.Logger
	executors map[string]MetricCollector
}

// NewController returns a new instance of Controller
func NewController(l *zap.Logger) *MetricCollectorController {
	return &MetricCollectorController{
		l:         l,
		executors: make(map[string]MetricCollector),
	}
}

// Collect will exec all collectors and return metric list.
func (c *MetricCollectorController) Collect(ctx context.Context) []*reporter.Metric {
	var res []*reporter.Metric

	for name, collector := range c.executors {
		c.l.Debug("Run metric collector", zap.String("collector", name))

		metricList, err := collector.Collect(ctx)
		if err != nil {
			c.l.Error(
				"Failed to collect metric",
				zap.String("collector", name),
				zap.Error(err),
			)
			continue
		}
		res = append(res, metricList...)
	}

	return res
}

// Register will register a metric collector.
func (c *MetricCollectorController) Register(metricName reporter.MetricName, collector MetricCollector) {
	c.executors[string(metricName)] = collector
}
