package v1

import (
	"context"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// validateStatementAnchor checks that a thread root's anchor names a spec in
// the issue's plan and a sheet the project can read, so the historical SQL
// the anchor promises is retrievable. The anchor is immutable, so this is
// the only chance; replies inherit the root's spec and sheet in the store.
func (s *IssueService) validateStatementAnchor(ctx context.Context, issue *store.IssueMessage, anchor *v1pb.StatementAnchor) error {
	if issue.PlanUID == nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("statement_anchor requires an issue with a plan"))
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: issue.ProjectID, UID: issue.PlanUID})
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get plan"))
	}
	if plan == nil || !slices.ContainsFunc(plan.Config.GetSpecs(), func(spec *storepb.PlanConfig_Spec) bool {
		return spec.Id == anchor.Spec
	}) {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("statement_anchor.spec %q is not a spec of the issue's plan", anchor.Spec))
	}
	missing, err := s.store.MissingSheetsForProject(ctx, issue.ProjectID, anchor.SheetSha256)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check sheet"))
	}
	if len(missing) > 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("statement_anchor.sheet_sha256 %q is not a sheet of project %s", anchor.SheetSha256, issue.ProjectID))
	}
	return nil
}

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
