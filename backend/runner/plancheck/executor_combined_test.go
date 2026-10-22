package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCombinedExecutorLeavesSQLReviewToV2(t *testing.T) {
	ctx := context.Background()
	s := setupPlancheckStore(ctx, t)
	for instanceID, engine := range map[string]storepb.Engine{
		"pg":     storepb.Engine_POSTGRES,
		"dynamo": storepb.Engine_DYNAMODB,
	} {
		_, err := s.CreateInstance(ctx, &store.InstanceMessage{
			ResourceID: instanceID,
			Workspace:  "default",
			Metadata: &storepb.Instance{
				Engine:      engine,
				DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
			},
		})
		require.NoError(t, err)
		_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
			ProjectID:    "project-a",
			InstanceID:   instanceID,
			DatabaseName: "app",
			Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
		})
		require.NoError(t, err)
	}
	coversPostgres := func(engine storepb.Engine) bool { return engine == storepb.Engine_POSTGRES }
	withV2 := NewCombinedExecutor(s, nil, nil, coversPostgres)
	withoutV2 := NewCombinedExecutor(s, nil, nil, nil)
	target := func(instanceID string) *CheckTarget {
		return &CheckTarget{Target: "instances/" + instanceID + "/databases/app"}
	}
	advise := storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE

	require.True(t, withV2.supersededBySQLReviewV2(ctx, target("pg"), advise))
	require.False(t, withV2.supersededBySQLReviewV2(ctx, target("dynamo"), advise), "V1 keeps advising an engine the rule review does not cover")
	require.False(t, withV2.supersededBySQLReviewV2(ctx, target("pg"), storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC), "no rule dry-runs gh-ost against the instance")
	require.False(t, withV2.supersededBySQLReviewV2(ctx, target("pg"), storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT))
	require.False(t, withoutV2.supersededBySQLReviewV2(ctx, target("pg"), advise))

	// Without a sheet manager, running the advice would panic.
	pg := target("pg")
	pg.Types = []storepb.PlanCheckType{advise}
	results, err := withV2.RunForTarget(ctx, pg)
	require.NoError(t, err)
	require.Empty(t, results)
}
