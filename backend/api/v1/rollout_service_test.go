package v1

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/review"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

// TestClassifyRolloutError pins the two answers a rollout caller acts on. The
// races that produce them — an approval reset while the rollout was being
// built, a draft that was never submitted — are pinned against the database
// in component/review.
func TestClassifyRolloutError(t *testing.T) {
	t.Parallel()
	workflowErr := func(reason review.ErrorReason) error {
		return &review.Error{Code: review.ErrorFailedPrecondition, Reason: reason, Err: errors.New("refused")}
	}
	require.ErrorIs(t, classifyRolloutError(workflowErr(review.ReasonDraftIssue)), errDraftIssueNotSubmitted)
	require.True(t, IsStaleRolloutApprovalError(classifyRolloutError(workflowErr(review.ReasonApprovalRequired))))
	require.True(t, IsStaleRolloutApprovalError(classifyRolloutError(workflowErr(review.ReasonStaleInput))))
	require.True(t, IsStaleRolloutApprovalError(classifyRolloutError(errors.Wrap(errStaleRolloutApproval, "building tasks"))))

	other := errors.New("failed to get pipeline create")
	require.Equal(t, other, classifyRolloutError(other), "anything else passes through as it came")
}
