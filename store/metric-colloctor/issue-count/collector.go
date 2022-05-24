package collector

import (
	"context"

	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

func init() {
	store.Register(metricAPI.IssueCountMetricName, collector{})
}

type collector struct{}

func (collector) Collect(ctx context.Context, l *zap.Logger, s *store.Store) ([]metric.Metric, error) {
	issueCountMetricList, err := s.CountIssueGroupByTypeAndStatus(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]metric.Metric, 0, len(issueCountMetricList))
	for _, issueCountMetric := range issueCountMetricList {
		res = append(res, issueCountMetric)
	}
	return res, nil
}
