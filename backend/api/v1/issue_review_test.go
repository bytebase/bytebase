package v1

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/component/review"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

func TestMapReviewErrorMarksOnlyTheApproverRoleVerdict(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		err      error
		wantCode connect.Code
		wantMark bool
	}{
		{
			name:     "approver role",
			err:      &review.Error{Code: review.ErrorPermissionDenied, Reason: review.ReasonApproverRoleRequired, Err: errors.New("cannot approve")},
			wantCode: connect.CodePermissionDenied,
			wantMark: true,
		},
		{
			name:     "self-approval or ownership",
			err:      &review.Error{Code: review.ErrorPermissionDenied, Err: errors.New("self-approval is not allowed")},
			wantCode: connect.CodePermissionDenied,
		},
		{
			name:     "issue state",
			err:      &review.Error{Code: review.ErrorInvalidAction, Err: errors.New("the issue has been approved")},
			wantCode: connect.CodeInvalidArgument,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			marked := false
			ctx := withSetPermissionDenied(context.Background(), func() { marked = true })

			err := mapReviewError(ctx, test.err, review.ActionApprove)

			require.Equal(t, test.wantCode, connect.CodeOf(err))
			require.Equal(t, test.wantMark, marked,
				"the audit interceptor streams a refusal, and stamps it WARNING, only when the request is marked")
		})
	}
}

func TestMapReviewErrorSaysWhenAConfirmedIssueChanged(t *testing.T) {
	t.Parallel()

	err := mapReviewError(context.Background(), &review.Error{
		Code:   review.ErrorConflict,
		Reason: review.ReasonChangedSinceConfirmed,
		Err:    errors.New("the issue changed after the action was confirmed"),
	}, review.ActionApprove)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "cannot approve because the issue changed after the decision was confirmed")

	err = mapReviewError(context.Background(), &review.Error{Code: review.ErrorConflict, Err: errors.New("approval finding is stale")}, review.ActionReject)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "cannot reject because approval finding is stale")
}

func TestConfirmedUpdateTimeComesFromTheMCPConfirmation(t *testing.T) {
	t.Parallel()

	require.Nil(t, confirmedUpdateTime(context.Background()), "a decision made in the console carries no confirmed version")
	at := time.Date(2026, 10, 13, 8, 0, 0, 123456000, time.UTC)
	ctx := auth.WithMCPConfirmedDecision(context.Background(), auth.MCPConfirmedDecision{
		Procedure:       v1connect.IssueServiceApproveIssueProcedure,
		Issue:           "projects/hr/issues/42",
		IssueUpdateTime: at,
	})
	require.Equal(t, &at, confirmedUpdateTime(ctx))
}
