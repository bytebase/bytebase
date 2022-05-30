package collector

import (
	"context"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
)

var _ metric.Collector = (*sheetCountCollector)(nil)

// sheetCountCollector is the metric data collector for sheet.
type sheetCountCollector struct {
	store *store.Store
}

// NewSheetCountCollector creates a new instance of sheetCountCollector
func NewSheetCountCollector(store *store.Store) metric.Collector {
	return &sheetCountCollector{
		store: store,
	}
}

// Collect will collect the metric for sheet
func (c *sheetCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	sheetCountMetricList, err := c.store.CountSheetGroupByVisibility(ctx)
	if err != nil {
		return nil, err
	}

	for _, sheetCountMetric := range sheetCountMetricList {
		res = append(res, &metric.Metric{
			Name:  metricAPI.SheetCountMetricName,
			Value: sheetCountMetric.Count,
			Labels: map[string]string{
				"visibility": string(sheetCountMetric.Visibility),
			},
		})
	}

	return res, nil
}
