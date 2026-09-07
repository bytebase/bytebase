package v1

import (
	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// issueCommentError maps a store error to the connect code the rest of the
// package uses for the same class: Conflict is FailedPrecondition, as in
// rollout_service.go and instance_service.go.
func issueCommentError(err error) error {
	code := connect.CodeInternal
	switch common.ErrorCode(err) {
	case common.Invalid:
		code = connect.CodeInvalidArgument
	case common.NotFound:
		code = connect.CodeNotFound
	case common.Conflict:
		code = connect.CodeFailedPrecondition
	default:
	}
	return connect.NewError(code, err)
}

func convertToStoreThreadState(state v1pb.IssueComment_ThreadState) (store.ThreadState, error) {
	switch state {
	case v1pb.IssueComment_OPEN:
		return store.ThreadStateOpen, nil
	case v1pb.IssueComment_RESOLVED:
		return store.ThreadStateResolved, nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("thread_state must be OPEN or RESOLVED"))
	}
}
