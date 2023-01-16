package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/tests/fake"
)

type trial struct {
	plan       api.PlanType
	expectPlan api.PlanType
	Days       int
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
			plan:       api.TEAM,
			expectPlan: api.TEAM,
			Days:       7,
		},
		{
			// Test trial the ENTERPRISE plan.
			plan:       api.ENTERPRISE,
			expectPlan: api.ENTERPRISE,
			Days:       7,
		},
		{
			// Downgrade should be ignored.
			plan:       api.TEAM,
			expectPlan: api.ENTERPRISE,
			Days:       7,
		},
	}

	for _, trial := range trialList {
		err = ctl.trialPlan(&api.TrialPlanCreate{
			Type: trial.plan,
			Days: trial.Days,
		})
		a.NoError(err)

		subscription, err = ctl.getSubscription()
		a.NoError(err)
		a.Equal(trial.expectPlan, subscription.Plan)
	}
}
