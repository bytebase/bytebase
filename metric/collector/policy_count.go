package collector

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
)

var _ metric.Collector = (*policyCountCollector)(nil)

// policyCountCollector is the metric data collector for policy.
type policyCountCollector struct {
	store *store.Store
}

// NewPolicyCountCollector creates a new instance of policyCollector
func NewPolicyCountCollector(store *store.Store) metric.Collector {
	return &policyCountCollector{
		store: store,
	}
}

// Collect will collect the metric for policy
func (c *policyCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	policyList, err := c.store.ListPolicy(ctx, &api.PolicyFind{})
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
			value = string(payload.Value)
			key = fmt.Sprintf("%s_%s_%s", policy.Type, policy.Environment.Name, value)
		case api.PolicyTypeBackupPlan:
			payload, err := api.UnmarshalBackupPlanPolicy(policy.Payload)
			if err != nil {
				continue
			}
			value = string(payload.Schedule)
			key = fmt.Sprintf("%s_%s_%s", policy.Type, policy.Environment.Name, value)
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
				Value:           value,
				EnvironmentName: policy.Environment.Name,
				Count:           0,
			}
		}
		policyCountMap[key].Count++
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
