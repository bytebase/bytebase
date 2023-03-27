package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type trial struct {
	seat                int32
	instanceCount       int32
	expectInstanceCount int32
	plan                v1pb.PlanType
	expectPlan          v1pb.PlanType
	Days                int32
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
	a.Equal(v1pb.PlanType_FREE, subscription.Plan)

	trialList := []trial{
		{
			// Test trial the TEAM plan.
			seat:                10,
			instanceCount:       20,
			expectInstanceCount: 20,
			plan:                v1pb.PlanType_TEAM,
			expectPlan:          v1pb.PlanType_TEAM,
			Days:                7,
		},
		{
			// Test trial the ENTERPRISE plan.
			seat:                10,
			instanceCount:       10,
			expectInstanceCount: 10,
			plan:                v1pb.PlanType_ENTERPRISE,
			expectPlan:          v1pb.PlanType_ENTERPRISE,
			Days:                7,
		},
		{
			// Downgrade should be ignored.
			seat:                10,
			instanceCount:       99,
			expectInstanceCount: 10,
			plan:                v1pb.PlanType_TEAM,
			expectPlan:          v1pb.PlanType_ENTERPRISE,
			Days:                7,
		},
	}

	for _, trial := range trialList {
		err = ctl.trialPlan(&v1pb.TrialSubscription{
			Seat:          trial.seat,
			InstanceCount: trial.instanceCount,
			Plan:          trial.plan,
			Days:          trial.Days,
		})
		a.NoError(err)

		subscription, err = ctl.getSubscription()
		a.NoError(err)
		a.Equal(trial.expectPlan, subscription.Plan)
		a.Equal(trial.expectInstanceCount, subscription.InstanceCount)
	}
}
