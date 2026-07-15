package review

import (
	"context"
	"slices"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// UpdateIssueLabelsInput identifies a semantic Bytebase Issue label change.
type UpdateIssueLabelsInput struct {
	Workspace string
	ProjectID string
	IssueUID  int64
	Labels    []string
}

// UpdateIssueLabelsResult is the committed label and review state.
type UpdateIssueLabelsResult struct {
	Issue         *store.IssueMessage
	ApprovalReset bool
	Events        []Event
}

// UpdateIssueLabels updates labels and any required approval reset atomically.
func (w *Workflow) UpdateIssueLabels(ctx context.Context, input UpdateIssueLabelsInput) (*UpdateIssueLabelsResult, error) {
	project, err := w.store.GetProject(ctx, &store.FindProjectMessage{
		Workspace:  input.Workspace,
		ResourceID: &input.ProjectID,
	})
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to get project")
	}
	if project == nil {
		return nil, workflowError(ErrorNotFound, "project %s not found", input.ProjectID)
	}

	tx, err := w.store.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to begin issue label transaction")
	}
	defer tx.Rollback()
	issue, err := lockIssue(ctx, tx, input.ProjectID, input.IssueUID)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, workflowError(ErrorNotFound, "issue %d not found in project %s", input.IssueUID, input.ProjectID)
	}
	labels := store.CanonicalizeIssueLabels(input.Labels)
	result := &UpdateIssueLabelsResult{Issue: issue}
	if slices.Equal(store.CanonicalizeIssueLabels(issue.Payload.GetLabels()), labels) {
		if err := tx.Commit(); err != nil {
			return nil, workflowWrap(ErrorInternal, err, "failed to commit issue label transaction")
		}
		return result, nil
	}

	plan, err := lockIssuePlan(ctx, tx, issue)
	if err != nil {
		return nil, err
	}
	resetApproval := shouldResetApprovalForLabels(issue, plan)
	payloadPatch := &storepb.Issue{Labels: labels}
	if resetApproval {
		payloadPatch.Approval = &storepb.IssuePayloadApproval{
			ApprovalInputVersion: plan.Config.GetApprovalInputVersion(),
		}
	}
	if err := updateIssuePayload(ctx, tx, issue, payloadPatch, len(labels) == 0); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to update issue labels")
	}
	issue.Payload.Labels = labels
	if resetApproval {
		issue.Payload.Approval = payloadPatch.Approval
		result.ApprovalReset = true
		result.Events = []Event{ApprovalCheckEvent{}}
	}
	if err := tx.Commit(); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to commit issue label transaction")
	}
	return result, nil
}

func shouldResetApprovalForLabels(issue *store.IssueMessage, plan *store.PlanMessage) bool {
	if issue.Payload.GetDraft() || issue.Type != storepb.Issue_DATABASE_CHANGE || plan == nil || plan.Config.GetHasRollout() {
		return false
	}
	for _, spec := range plan.Config.GetSpecs() {
		if _, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig); ok {
			return true
		}
	}
	return false
}
