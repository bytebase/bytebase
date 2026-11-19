package reviewrun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCompletingAReviewRunRequestsARolloutCheck(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		sqlReviewV2 bool
	}{
		{name: "under SQL Review V2", sqlReviewV2: true},
		{name: "beside SQL Review V1, where open threads hold the rollout", sqlReviewV2: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			stores, issues := setupReviewFixture(ctx, t, 1)
			issue, err := stores.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{"p"}, UID: &issues[0]})
			require.NoError(t, err)
			require.NotNil(t, issue.PlanUID)
			b, err := bus.New()
			require.NoError(t, err)
			scheduler := &Scheduler{store: stores, bus: b, profile: &config.Profile{SQLReviewV2: test.sqlReviewV2, ReplicaID: "replica-1"}}
			_, err = stores.CreateReviewRun(ctx, "p", issues[0])
			require.NoError(t, err)
			claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica-1")
			require.NoError(t, err)
			require.Len(t, claimed, 1)

			scheduler.completeReviewRun(ctx, claimed[0], nil, nil)

			select {
			case ref := <-b.RolloutCreationChan:
				require.Equal(t, bus.PlanRef{ProjectID: "p", PlanID: *issue.PlanUID}, ref)
			default:
				require.Fail(t, "completing the run requested no rollout check")
			}
		})
	}
}
