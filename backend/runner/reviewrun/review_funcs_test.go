package reviewrun

import (
	"context"
	"testing"

	"github.com/bytebase/omni/review"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
)

func TestReviewFuncsEvaluateRules(t *testing.T) {
	t.Parallel()
	for engine, reviewFunc := range reviewFuncs {
		result, err := reviewFunc(context.Background(), "DELETE FROM t;", review.Options{Rules: []review.Rule{review.RequireWhere}}, []review.Target{{}})
		require.NoError(t, err, engine)
		require.Len(t, result.Findings, 1, engine)
		require.Equal(t, review.RequireWhere, result.Findings[0].Rule, engine)
		require.Equal(t, []int{0}, result.Findings[0].Targets, engine)
	}
}

func TestAIReviewStandardRulesFollowTheEngine(t *testing.T) {
	t.Parallel()
	policy := &storepb.ReviewRulePolicy{Rules: []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_WALK_THROUGH}}
	reviewed := func(engine storepb.Engine) []bool {
		var got []bool
		for _, rule := range aiReviewStandardRules(policy, engine) {
			require.Equal(t, aireview.SeverityP0, rule.Severity, rule.Type)
			got = append(got, rule.Reviewed)
		}
		return got
	}
	require.Equal(t, []bool{true, false}, reviewed(storepb.Engine_POSTGRES), "the walk can stop early, so the model checks WALK_THROUGH too")
	require.Equal(t, []bool{false, false}, reviewed(storepb.Engine_CLICKHOUSE), "no standard review runs on ClickHouse")
	require.True(t, reviewsRule(storepb.Engine_POSTGRES, storepb.ReviewRuleType_WALK_THROUGH))
	require.False(t, reviewsRule(storepb.Engine_SNOWFLAKE, storepb.ReviewRuleType_WALK_THROUGH), "omni's Snowflake review does not walk through")
}
