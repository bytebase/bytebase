package reviewrun

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/code"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/store"
)

// LicenseChecker reports why a workspace's plan lacks a feature, or nil when
// it has it. enterprise.LicenseService implements it.
type LicenseChecker interface {
	IsFeatureEnabled(ctx context.Context, workspaceID string, f v1pb.PlanFeature) error
}

// ReviewExecutor runs an issue's review as one job: the standard rules, then
// the AI review while the workspace allows it. The results of both post
// together, or on an error none does. The rules run first, so their failure
// ends the job before any model call, and the AI review's problems on the
// operations the rules flag join the rules' results.
type ReviewExecutor struct {
	store   *store.Store
	license LicenseChecker
	// rule is nil where SQL Review V2 is off: no standard rule review runs
	// there, and the review is the AI review alone.
	rule ruleReviewer
	ai   aiReviewer
}

// ruleReviewer is the standard rules' part of a review.
type ruleReviewer interface {
	// review returns the rules' results before they become comments.
	review(ctx context.Context, projectID string, issueUID int64) ([]*ruleResult, error)
}

// aiReviewer is the AI review's part of a review.
type aiReviewer interface {
	// reviewWith returns the results of the whole review to post: the rules'
	// results, with what the AI review adds to them, and its own.
	reviewWith(ctx context.Context, projectID string, issueUID int64, rules []*ruleResult) ([]*store.IssueCommentMessage, error)
}

// NewReviewExecutor creates the executor of the review. rule is nil where SQL
// Review V2 is off.
func NewReviewExecutor(s *store.Store, license LicenseChecker, rule *RuleExecutor, ai *AIReviewExecutor) *ReviewExecutor {
	e := &ReviewExecutor{store: s, license: license, ai: ai}
	// A nil *RuleExecutor in the interface would not compare equal to nil.
	if rule != nil {
		e.rule = rule
	}
	return e
}

// RunOnce implements Executor.
func (e *ReviewExecutor) RunOnce(ctx context.Context, projectID string, issueUID int64) ([]*store.IssueCommentMessage, error) {
	plan, project, err := loadReviewPlan(ctx, e.store, projectID, issueUID)
	if err != nil {
		return nil, err
	}

	var rules []*ruleResult
	if e.rule != nil {
		rules, err = e.rule.review(ctx, projectID, issueUID)
		if err != nil {
			return nil, err
		}
	}
	// The rules' results as they post without the AI review.
	ruleResults := func() []*store.IssueCommentMessage {
		results := ruleResultComments(projectID, issueUID, rules)
		sortReviewResults(plan.Config.GetSpecs(), results)
		return results
	}

	aiOn, err := e.aiReviewOn(ctx, project.Workspace)
	if err != nil {
		return nil, err
	}
	if !aiOn {
		return ruleResults(), nil
	}
	results, err := e.ai.reviewWith(ctx, projectID, issueUID, rules)
	if err != nil {
		if ctx.Err() != nil {
			return nil, err
		}
		// Switched off while it ran: a reviewer that is off is skipped, so the
		// review ends with the rules' results.
		if aiOn, settingErr := e.aiReviewOn(ctx, project.Workspace); settingErr == nil && !aiOn {
			return ruleResults(), nil
		}
		return nil, err
	}
	sortReviewResults(plan.Config.GetSpecs(), results)
	return results, nil
}

// aiReviewOn reports whether the workspace AI setting and the workspace's
// plan allow the AI review. It reads past the setting cache, as the AI review
// does.
func (e *ReviewExecutor) aiReviewOn(ctx context.Context, workspace string) (bool, error) {
	setting, err := e.store.GetAISettingUncached(ctx, workspace)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get AI setting")
	}
	return aireview.CheckSetting(setting) == nil &&
		e.license.IsFeatureEnabled(ctx, workspace, v1pb.PlanFeature_FEATURE_AI_SQL_REVIEW) == nil, nil
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
