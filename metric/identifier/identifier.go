package identifier

import (
	"context"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
)

type metricIdentifier struct {
	store     *store.Store
	workspace *api.Workspace
	// subscription is the pointer to the server.subscription.
	// the subscription can be updated by users so we need the pointer to get the latest value.
	subscription *enterpriseAPI.Subscription
}

const (
	// identifyTraitForPlan is the trait key for subscription plan.
	identifyTraitForPlan = "plan"
	// identifyTraitForVersion is the trait key for bytebase version.
	identifyTraitForVersion = "version"
	// identifyTraitForActiveTaskCount is the trait key for active task count in the workspace.
	identifyTraitForActiveTaskCount = "active_task_count"
)

// NewIdentifier creates a new instance of metricIdentifier
func NewIdentifier(store *store.Store, workspace *api.Workspace, subscription *enterpriseAPI.Subscription) metric.Identifier {
	return &metricIdentifier{
		store:        store,
		workspace:    workspace,
		subscription: subscription,
	}
}

// Identify returns the metric Identity
func (i *metricIdentifier) Identify(ctx context.Context) (*metric.Identity, error) {
	plan := api.FREE.String()
	if i.subscription != nil {
		plan = i.subscription.Plan.String()
	}

	now := time.Now()
	from := now.AddDate(0, 0, -7)
	count, err := i.store.CountTaskInRangeOfTimeByStatus(ctx, from.Unix(), now.Unix(), api.TaskDone)
	if err != nil {
		return nil, err
	}

	return &metric.Identity{
		ID: i.workspace.ID,
		Labels: map[string]string{
			identifyTraitForPlan:            plan,
			identifyTraitForVersion:         i.workspace.Version,
			identifyTraitForActiveTaskCount: strconv.Itoa(count),
		},
	}, nil
}
