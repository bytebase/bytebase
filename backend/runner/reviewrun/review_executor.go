package reviewrun

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/code"

	"github.com/bytebase/bytebase/backend/plugin/ai"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/store"
)

// ReviewExecutor runs an issue's review as one job: the standard rules, then
// the AI review while the workspace allows it. The results of both post
// together, or on an error none does. The rules run first, so their failure
// ends the job before any model call.
type ReviewExecutor struct {
	store *store.Store
	// rule is nil where SQL Review V2 is off: no standard rule review runs
	// there, and the review is the AI review alone.
	rule Executor
	ai   Executor
}

// NewReviewExecutor creates the executor of the review. rule is nil where SQL
// Review V2 is off.
func NewReviewExecutor(s *store.Store, rule, ai Executor) *ReviewExecutor {
	return &ReviewExecutor{store: s, rule: rule, ai: ai}
}

// RunOnce implements Executor.
func (e *ReviewExecutor) RunOnce(ctx context.Context, projectID string, issueUID int64) ([]*store.IssueCommentMessage, error) {
	plan, project, err := loadReviewPlan(ctx, e.store, projectID, issueUID)
	if err != nil {
		return nil, err
	}

	var results []*store.IssueCommentMessage
	if e.rule != nil {
		results, err = e.rule.RunOnce(ctx, projectID, issueUID)
		if err != nil {
			return nil, err
		}
	}

	aiOn, err := e.aiReviewOn(ctx, project.Workspace)
	if err != nil {
		return nil, err
	}
	if !aiOn {
		return results, nil
	}
	aiResults, err := e.ai.RunOnce(ctx, projectID, issueUID)
	if err != nil {
		if ctx.Err() != nil {
			return nil, err
		}
		// Switched off while it ran: a reviewer that is off is skipped, so the
		// review ends with the rules' results.
		if aiOn, settingErr := e.aiReviewOn(ctx, project.Workspace); settingErr == nil && !aiOn {
			return results, nil
		}
		return nil, err
	}

	results = append(results, aiResults...)
	sortReviewResults(plan.Config.GetSpecs(), results)
	return results, nil
}

// aiReviewOn reports whether the workspace AI setting allows the AI review.
// It reads past the setting cache, as the AI review does.
func (e *ReviewExecutor) aiReviewOn(ctx context.Context, workspace string) (bool, error) {
	setting, err := e.store.GetAISettingUncached(ctx, workspace)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get AI setting")
	}
	return aireview.CheckSetting(setting) == nil, nil
}

// errorCode is the canonical code of the failure that ended a run, for a
// reader that acts on the kind of failure. A provider's refusal anywhere in
// the failure decides the code. UNKNOWN is a failure no code here describes.
func errorCode(err error) code.Code {
	var httpErr *ai.HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.StatusCode == http.StatusTooManyRequests:
			return code.Code_RESOURCE_EXHAUSTED
		case httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode == http.StatusForbidden:
			// The provider refuses the workspace's own key.
			return code.Code_FAILED_PRECONDITION
		case httpErr.StatusCode >= http.StatusInternalServerError:
			return code.Code_UNAVAILABLE
		default:
			return code.Code_UNKNOWN
		}
	}
	var transportErr *ai.TransportError
	switch {
	case errors.As(err, &transportErr):
		return code.Code_UNAVAILABLE
	case errors.Is(err, aireview.ErrDeadline), errors.Is(err, context.DeadlineExceeded):
		return code.Code_DEADLINE_EXCEEDED
	case errors.Is(err, errMetadataNotSynced):
		return code.Code_FAILED_PRECONDITION
	default:
		return code.Code_UNKNOWN
	}
}
