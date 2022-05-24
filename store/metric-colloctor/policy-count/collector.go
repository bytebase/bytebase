package collector

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

func init() {
	store.Register(metricAPI.PolicyCountMetricName, collector{})
}

type collector struct{}

func (collector) Collect(ctx context.Context, l *zap.Logger, s *store.Store) ([]metric.Metric, error) {
	var res []metric.Metric

	policyList, err := s.ListPolicy(ctx, &api.PolicyFind{})
	if err != nil {
		return nil, err
	}

	policyCountMap := make(map[string]*metricAPI.PolicyCountMetric)

	for _, policy := range policyList {
		var key string
		var value string

		switch policy.Type {
		case api.PolicyTypePipelineApproval:
			payload, err := api.UnmarshalPipelineApprovalPolicy(policy.Payload)
			if err != nil {
				continue
			}
			key = fmt.Sprintf("%s_%s_%s", policy.Type, policy.Environment.Name, value)
			value = string(payload.Value)
		case api.PolicyTypeBackupPlan:
			payload, err := api.UnmarshalBackupPlanPolicy(policy.Payload)
			if err != nil {
				continue
			}
			key = fmt.Sprintf("%s_%s_%s", policy.Type, policy.Environment.Name, value)
			value = string(payload.Schedule)
		case api.PolicyTypeSchemaReview:
			key = fmt.Sprintf("%s_%s", policy.Type, policy.Environment.Name)
			// schema review policy don't need to set the value.
			value = ""
		}

		if key == "" {
			continue
		}

		if _, ok := policyCountMap[key]; !ok {
			policyCountMap[key] = &metricAPI.PolicyCountMetric{
				Type:            policy.Type,
				Payload:         value,
				EnvironmentName: policy.Environment.Name,
				Count:           0,
			}
		}
		policyCountMap[key].Count++
	}

	for _, policyCountMetric := range policyCountMap {
		res = append(res, policyCountMetric)
	}

	return res, nil
}
