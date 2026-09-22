package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/review"
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
