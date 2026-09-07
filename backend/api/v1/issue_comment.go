package v1

import (
	"context"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// Only new roots require a current spec; replies may reference a deleted spec.
func (s *IssueService) validateStatementAnchorSpec(ctx context.Context, issue *store.IssueMessage, anchor *v1pb.StatementAnchor) error {
	if issue.PlanUID == nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("statement_anchor requires an issue with a plan"))
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: issue.ProjectID, UID: issue.PlanUID})
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get plan"))
	}
	if plan != nil {
		for _, spec := range plan.Config.GetSpecs() {
			if spec.GetId() != anchor.Spec {
				continue
			}
			if spec.GetChangeDatabaseConfig() == nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("statement_anchor.spec %q does not have a statement-bearing configuration", anchor.Spec))
			}
			return nil
		}
	}
	return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("statement_anchor.spec %q is not a spec of the issue's plan", anchor.Spec))
}

func (s *IssueService) validateStatementAnchor(ctx context.Context, issue *store.IssueMessage, anchor *v1pb.StatementAnchor) error {
	sheet, err := s.store.GetSheetForProject(ctx, issue.ProjectID, anchor.SheetSha256, true)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get anchor sheet"))
	}
	if sheet == nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("statement_anchor.sheet_sha256 %q is not a sheet of project %s", anchor.SheetSha256, issue.ProjectID))
	}
	if err := validateStatementAnchorBounds(anchor, sheet.Statement); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil
}

// The store validates range ordering and reply containment; both endpoints
// must also exist in the saved SQL, even for replies inside a whole-line root.
func validateStatementAnchorBounds(anchor *v1pb.StatementAnchor, statement string) error {
	start, end := anchor.GetStartPosition(), anchor.GetEndPosition()
	if start == nil || end == nil || start.Line < 1 || end.Line < 1 || start.Column < 0 || end.Column < 0 {
		return errors.New("statement_anchor requires valid start and end positions")
	}
	// Match editor line endings and retain the empty line after a final newline.
	statement = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(statement)
	lineNumber := int64(0)
	for line := range strings.SplitSeq(statement, "\n") {
		lineNumber++
		if lineNumber == int64(start.Line) || lineNumber == int64(end.Line) {
			maxColumn := int64(utf8.RuneCountInString(line)) + 1
			for _, position := range []*v1pb.Position{start, end} {
				if int64(position.Line) == lineNumber && int64(position.Column) > maxColumn {
					return errors.Errorf("statement_anchor column %d exceeds line %d's last position %d", position.Column, position.Line, maxColumn)
				}
			}
		}
		if lineNumber >= int64(start.Line) && lineNumber >= int64(end.Line) {
			return nil
		}
	}
	return errors.Errorf("statement_anchor range exceeds the sheet's %d lines", lineNumber)
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
