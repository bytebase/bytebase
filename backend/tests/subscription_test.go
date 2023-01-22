package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/tests/fake"
)

type trial struct {
	instanceCount       int
	expectInstanceCount int
	plan                api.PlanType
	expectPlan          api.PlanType
	Days                int
}

func TestSubscription(t *testing.T) {
	a := require.New(t)
	log.SetLevel(zapcore.DebugLevel)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	subscription, err := ctl.getSubscription()
	a.NoError(err)
	a.Equal(api.FREE, subscription.Plan)

	trialList := []trial{
		{
			// Test trial the TEAM plan.
			instanceCount:       20,
			expectInstanceCount: 20,
			plan:                api.TEAM,
			expectPlan:          api.TEAM,
			Days:                7,
		},
		{
			// Test trial the ENTERPRISE plan.
			instanceCount:       10,
			expectInstanceCount: 10,
			plan:                api.ENTERPRISE,
			expectPlan:          api.ENTERPRISE,
			Days:                7,
		},
		{
			// Downgrade should be ignored.
			instanceCount:       99,
			expectInstanceCount: 10,
			plan:                api.TEAM,
			expectPlan:          api.ENTERPRISE,
			Days:                7,
		},
	}

	for _, trial := range trialList {
		err = ctl.trialPlan(&api.TrialPlanCreate{
			InstanceCount: trial.instanceCount,
			Type:          trial.plan,
			Days:          trial.Days,
		})
		a.NoError(err)

		subscription, err = ctl.getSubscription()
		a.NoError(err)
		a.Equal(trial.expectPlan, subscription.Plan)
		a.Equal(trial.expectInstanceCount, subscription.InstanceCount)
	}
}
