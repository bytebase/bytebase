package store

import (
	"context"
	"sync"

	"github.com/bytebase/bytebase/plugin/metric"
	"go.uber.org/zap"
)

// MetricCollector is the interface that must be implemented by a store metric collector.
type MetricCollector interface {
	Collect(context.Context, *zap.Logger, *Store) ([]metric.Metric, error)
}

var (
	collectorMu sync.RWMutex
	collectors  = make(map[metric.Name]MetricCollector)
)

// Register make a metric collector factory available by the provided name.
// If Register is called twice with the same name or if factory is nil,
// it panics.
func Register(name metric.Name, collector MetricCollector) {
	collectorMu.Lock()
	defer collectorMu.Unlock()
	if collector == nil {
		panic("store/metric: Register collector factory is nil")
	}
	if _, dup := collectors[name]; dup {
		panic("store/metric: Register called twice for collector " + name)
	}
	collectors[name] = collector
}

// Metrics returns a list of the metric names of the registered collectors.
func Metrics() []metric.Name {
	collectorMu.RLock()
	defer collectorMu.RUnlock()
	list := make([]metric.Name, 0, len(collectors))
	for name := range collectors {
		list = append(list, name)
	}
	return list
}

// Collectors returns a list of the registered collectors.
func Collectors() map[metric.Name]MetricCollector {
	collectorMu.RLock()
	defer collectorMu.RUnlock()
	list := make(map[metric.Name]MetricCollector, len(collectors))
	for name, collector := range collectors {
		list[name] = collector
	}
	return list
}
