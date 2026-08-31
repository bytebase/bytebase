package review

import (
	"context"
	"database/sql"
	"slices"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// UpdateIssueMetadataInput identifies a semantic Bytebase Issue metadata patch.
type UpdateIssueMetadataInput struct {
	Workspace string
	ProjectID string
	IssueUID  int64

	Title       *string
	Description *string
	Labels      *[]string
	// ExpectedDraft is the lifecycle state observed by the adapter. When set,
	// the mutation conflicts instead of changing semantics after a race.
	ExpectedDraft *bool
}

// UpdateIssueMetadataResult is the committed metadata and review state.
type UpdateIssueMetadataResult struct {
	Issue         *store.IssueMessage
	ApprovalReset bool
	Events        []Event
}

// UpdateIssueMetadata updates Issue metadata and any required approval reset atomically.
func (w *Workflow) UpdateIssueMetadata(ctx context.Context, input UpdateIssueMetadataInput) (*UpdateIssueMetadataResult, error) {
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

	var result *UpdateIssueMetadataResult
	err = w.store.RunActiveProjectAndInstancesLifecycleWrite(ctx, input.ProjectID, nil, func(tx *sql.Tx) error {
		issue, err := lockIssue(ctx, tx, input.ProjectID, input.IssueUID)
		if err != nil {
			return err
		}
		if issue == nil {
			return workflowError(ErrorNotFound, "issue %d not found in project %s", input.IssueUID, input.ProjectID)
		}
		if input.ExpectedDraft != nil && issue.Payload.GetDraft() != *input.ExpectedDraft {
			return workflowError(ErrorConflict, "issue draft state changed while metadata was being updated")
		}
		result = &UpdateIssueMetadataResult{Issue: issue}
		patch := &store.UpdateIssueMessage{}
		if input.Title != nil && *input.Title != issue.Title {
			patch.Title = input.Title
			result.Events = append(result.Events, IssueTitleUpdatedEvent{FromTitle: issue.Title, ToTitle: *input.Title})
		}
		if input.Description != nil && *input.Description != issue.Description {
			patch.Description = input.Description
			result.Events = append(result.Events, IssueDescriptionUpdatedEvent{FromDescription: issue.Description, ToDescription: *input.Description})
		}

		var labels []string
		oldLabels := store.CanonicalizeIssueLabels(issue.Payload.GetLabels())
		labelsChanged := false
		if input.Labels != nil {
			labels = store.CanonicalizeIssueLabels(*input.Labels)
			labelsChanged = !slices.Equal(oldLabels, labels)
		}
		if !labelsChanged && patch.Title == nil && patch.Description == nil {
			return nil
		}

		if labelsChanged {
			var plan *store.PlanMessage
			if !issue.Payload.GetDraft() && issue.Type == storepb.Issue_DATABASE_CHANGE && issue.PlanUID != nil {
				plan, err = lockIssuePlan(ctx, tx, issue)
				if err != nil {
					return err
				}
			}
			resetApproval := shouldResetApprovalForLabels(issue, plan)
			payloadPatch := &storepb.Issue{Labels: labels}
			if resetApproval {
				payloadPatch.Approval = &storepb.IssuePayloadApproval{ApprovalInputVersion: plan.Config.GetApprovalInputVersion()}
				result.ApprovalReset = true
			}
			patch.PayloadUpsert = payloadPatch
			patch.RemoveLabels = len(labels) == 0
			result.Events = append(result.Events, IssueLabelsUpdatedEvent{FromLabels: oldLabels, ToLabels: labels})
			if resetApproval {
				result.Events = append(result.Events, ApprovalCheckEvent{})
			}
		}
		updatedAt, err := store.UpdateIssueTx(ctx, tx, issue, patch)
		if err != nil {
			return workflowWrap(ErrorInternal, err, "failed to update issue metadata")
		}
		issue.UpdatedAt = updatedAt
		if patch.Title != nil {
			issue.Title = *patch.Title
		}
		if patch.Description != nil {
			issue.Description = *patch.Description
		}
		if labelsChanged {
			issue.Payload.Labels = labels
			if result.ApprovalReset {
				issue.Payload.Approval = patch.PayloadUpsert.Approval
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
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
