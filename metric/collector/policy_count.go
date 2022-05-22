package collector

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/metric/collector"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// policyCollector is the metric data collector for policy.
type policyCollector struct {
	l     *zap.Logger
	store *store.Store
}

// NewPolicyCollector creates a new instance of policyCollector
func NewPolicyCollector(l *zap.Logger, store *store.Store) collector.MetricCollector {
	return &policyCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the metric for policy
func (c *policyCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	policyList, err := c.store.ListPolicy(ctx, &api.PolicyFind{})
	if err != nil {
		return nil, err
	}

	policyCountMap := make(map[string]metricAPI.PolicyCountMetric)

	for _, policy := range policyList {
		key := fmt.Sprintf("%s_%s", policy.Type, policy.Environment.Name)
		value := ""
		switch policy.Type {
		case api.PolicyTypePipelineApproval:
			payload, err := api.UnmarshalPipelineApprovalPolicy(policy.Payload)
			if err != nil {
				continue
			}
			value = string(payload.Value)
			key = fmt.Sprintf("%s_%s", key, value)
		case api.PolicyTypeBackupPlan:
			payload, err := api.UnmarshalBackupPlanPolicy(policy.Payload)
			if err != nil {
				continue
			}
			value = string(payload.Schedule)
			key = fmt.Sprintf("%s_%s", key, value)
		case api.PolicyTypeSchemaReview:
			// schema review policy don't need to set the value.
			value = ""
		}

		policyCountMetric, ok := policyCountMap[key]
		if !ok {
			policyCountMetric = metricAPI.PolicyCountMetric{
				Type:            policy.Type,
				Value:           value,
				EnvironmentName: policy.Environment.Name,
				Count:           0,
			}
		}
		policyCountMetric.Count++
		policyCountMap[key] = policyCountMetric
	}

	for _, policyCountMetric := range policyCountMap {
		res = append(res, &metric.Metric{
			Name:  metricAPI.PolicyCountMetricName,
			Value: policyCountMetric.Count,
			Labels: map[string]string{
				"type":        string(policyCountMetric.Type),
				"environment": policyCountMetric.EnvironmentName,
				"value":       policyCountMetric.Value,
			},
		})
	}

	return res, nil
}
