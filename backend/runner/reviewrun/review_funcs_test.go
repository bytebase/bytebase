package reviewrun

import (
	"context"
	"testing"

	"github.com/bytebase/omni/review"
	"github.com/stretchr/testify/require"
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
