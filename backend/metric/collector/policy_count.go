package collector

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*policyCountCollector)(nil)

// policyCountCollector is the metric data collector for policy.
type policyCountCollector struct {
	store *store.Store
}

// NewPolicyCountCollector creates a new instance of policyCollector.
func NewPolicyCountCollector(store *store.Store) metric.Collector {
	return &policyCountCollector{
		store: store,
	}
}

// Collect will collect the metric for policy.
func (c *policyCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	policies, err := c.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{})
	if err != nil {
		return nil, err
	}

	policyCountMap := make(map[string]*metricapi.PolicyCountMetric)

	for _, policy := range policies {
		var key string
		var value string
		if policy.ResourceType != api.PolicyResourceTypeEnvironment {
			continue
		}
		environmentID, err := common.GetEnvironmentID(policy.Resource)
		if err != nil {
			return nil, err
		}
		environment, err := c.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
		if err != nil {
			continue
		}
		if environment == nil {
			continue
		}

		rowStatus := api.Normal
		if environment.Deleted {
			rowStatus = api.Archived
		}

		if key == "" {
			continue
		}

		if _, ok := policyCountMap[key]; !ok {
			policyCountMap[key] = &metricapi.PolicyCountMetric{
				Type:            policy.Type,
				Value:           value,
				EnvironmentName: environment.Title,
				Count:           0,
				RowStatus:       rowStatus,
			}
		}
		policyCountMap[key].Count++
	}

	for _, policyCountMetric := range policyCountMap {
		res = append(res, &metric.Metric{
			Name:  metricapi.PolicyCountMetricName,
			Value: policyCountMetric.Count,
			Labels: map[string]any{
				"type":        string(policyCountMetric.Type),
				"environment": policyCountMetric.EnvironmentName,
				"value":       policyCountMetric.Value,
				"status":      string(policyCountMetric.RowStatus),
			},
		})
	}

	return res, nil
}
