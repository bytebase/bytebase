package plancheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
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

func TestCombinedExecutorHandsSQLReviewToV1WhereEveryRuleIsOff(t *testing.T) {
	ctx := context.Background()
	s := setupPlancheckStore(ctx, t)
	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "pg",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "project-a",
		InstanceID:   "pg",
		DatabaseName: "app",
		Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
	})
	require.NoError(t, err)
	setRules := func(resourceType storepb.Policy_Resource, resource string, rules ...storepb.ReviewRuleType) {
		payload, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: rules})
		require.NoError(t, err)
		_, err = s.CreatePolicy(ctx, &store.PolicyMessage{
			Workspace:    "default",
			ResourceType: resourceType,
			Resource:     resource,
			Type:         storepb.Policy_REVIEW_RULE,
			Payload:      string(payload),
			Enforce:      true,
		})
		require.NoError(t, err)
	}
	executor := NewCombinedExecutor(s, nil, nil, func(storepb.Engine) bool { return true })
	target := &CheckTarget{Target: "instances/pg/databases/app"}
	advise := storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE

	setRules(storepb.Policy_WORKSPACE, common.FormatWorkspace("default"))
	require.False(t, executor.supersededBySQLReviewV2(ctx, target, advise))

	setRules(storepb.Policy_PROJECT, common.FormatProject("project-a"), storepb.ReviewRuleType_SYNTAX)
	require.True(t, executor.supersededBySQLReviewV2(ctx, target, advise), "the project's own policy wins over the workspace's")

	setRules(storepb.Policy_PROJECT, common.FormatProject("project-a"), storepb.ReviewRuleType_REQUIRE_WHERE)
	require.False(t, executor.supersededBySQLReviewV2(ctx, target, advise), "without SYNTAX no rule runs")
}
