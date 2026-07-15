package review

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// CreateRolloutInput identifies a reviewed Plan ready for task creation.
type CreateRolloutInput struct {
	Workspace  string
	ProjectID  string
	PlanUID    int64
	BuildTasks RolloutTaskBuilder
}

// RolloutTaskBuilder computes tasks from the workflow's current Plan snapshot.
type RolloutTaskBuilder func(context.Context, *store.PlanMessage, *store.ProjectMessage) ([]*store.TaskMessage, error)

// CreateRolloutResult is the committed rollout state and its follow-up effects.
type CreateRolloutResult struct {
	Plan    *store.PlanMessage
	Issue   *store.IssueMessage
	Project *store.ProjectMessage
	Tasks   []*store.TaskMessage
	Events  []Event
}

// CreateRollout validates current review state and atomically marks the Plan and creates tasks.
func (w *Workflow) CreateRollout(ctx context.Context, input CreateRolloutInput) (*CreateRolloutResult, error) {
	var project *store.ProjectMessage
	var err error
	if input.Workspace == "" {
		project, err = w.store.GetProjectByResourceID(ctx, input.ProjectID)
	} else {
		project, err = w.store.GetProject(ctx, &store.FindProjectMessage{
			Workspace:  input.Workspace,
			ResourceID: &input.ProjectID,
		})
	}
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to get project")
	}
	if project == nil {
		return nil, workflowError(ErrorNotFound, "project %s not found", input.ProjectID)
	}
	plan, err := w.store.GetPlan(ctx, &store.FindPlanMessage{
		Workspace: project.Workspace,
		ProjectID: input.ProjectID,
		UID:       &input.PlanUID,
	})
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to get plan")
	}
	if plan == nil {
		return nil, workflowError(ErrorNotFound, "plan %d not found", input.PlanUID)
	}
	issue, err := w.store.GetIssue(ctx, &store.FindIssueMessage{
		Workspace:  project.Workspace,
		ProjectIDs: []string{input.ProjectID},
		PlanUID:    &input.PlanUID,
	})
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to get linked issue")
	}
	if issue != nil && issue.Payload.GetDraft() {
		return nil, workflowReasonError(ErrorFailedPrecondition, ReasonDraftIssue, "draft issue must be submitted before rollout creation")
	}
	if project.Setting.GetRequireIssueApproval() && issue != nil {
		approved, err := utils.CheckIssueApprovedForPlan(issue, plan)
		if err != nil {
			return nil, workflowWrap(ErrorInternal, err, "failed to check issue approval")
		}
		if !approved {
			return nil, workflowReasonError(ErrorFailedPrecondition, ReasonApprovalRequired, "cannot create rollout because issue approval is required but the issue is not approved")
		}
	}
	if input.BuildTasks == nil {
		return nil, workflowError(ErrorInternal, "rollout task builder is required")
	}
	tasks, err := input.BuildTasks(ctx, plan, project)
	if err != nil {
		return nil, err
	}

	guard := &store.RolloutGuard{ApprovalInputVersion: plan.Config.GetApprovalInputVersion()}
	if issue != nil && issue.Type == storepb.Issue_DATABASE_CHANGE && project.Setting.GetRequireIssueApproval() {
		guard.IssueUID = issue.UID
		guard.Approval = proto.CloneOf(issue.Payload.GetApproval())
	}
	marked, tasks, err := w.store.CreateRolloutTasks(ctx, input.ProjectID, input.PlanUID, guard, tasks)
	if err != nil {
		if errors.Is(err, store.ErrDraftIssueNotSubmitted) {
			return nil, workflowError(ErrorFailedPrecondition, "draft issue must be submitted before rollout creation")
		}
		return nil, workflowWrap(ErrorInternal, err, "failed to create rollout tasks")
	}
	if !marked {
		return nil, workflowReasonError(ErrorConflict, ReasonStaleInput, "issue approval is stale")
	}
	result := &CreateRolloutResult{Plan: plan, Issue: issue, Project: project, Tasks: tasks}
	if issue != nil {
		result.Events = []Event{CompleteRolloutIssueEvent{}}
	}
	return result, nil
}
