package collector

import (
	"context"

	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

var _ metric.Collector = (*memberCountCollector)(nil)

// memberCountCollector is the metric data collector for member.
type memberCountCollector struct {
	store *store.Store
}

// NewMemberCountCollector creates a new instance of memberCountCollector.
func NewMemberCountCollector(store *store.Store) metric.Collector {
	return &memberCountCollector{
		store: store,
	}
}

// Collect will collect the metric for issue.
func (c *memberCountCollector) Collect(ctx context.Context) ([]*metric.Metric, error) {
	var res []*metric.Metric

	workspaceIAMPolicy, err := c.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, err
	}

	roleMap := map[string]int{}
	for _, binding := range workspaceIAMPolicy.Policy.Bindings {
		roleMap[binding.Role] += len(binding.Members)
	}

	for role, count := range roleMap {
		res = append(res, &metric.Metric{
			Name:  metricapi.MemberCountMetricName,
			Value: count,
			Labels: map[string]any{
				"role": role,
			},
		})
	}

	return res, nil
}
