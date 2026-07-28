package review

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// UpdatePlanInput describes a Plan mutation. The workflow keeps a linked draft
// issue's metadata and lifecycle synchronized with the Plan and resets review
// state when submitted Plan specs change.
type UpdatePlanInput struct {
	Workspace   string
	ProjectID   string
	PlanUID     int64
	Title       *string
	Description *string
	Deleted     *bool
	Specs       *[]*storepb.PlanConfig_Spec
}

// UpdatePlanResult is the committed Plan and linked review state.
type UpdatePlanResult struct {
	Plan          *store.PlanMessage
	Issue         *store.IssueMessage
	ApprovalReset bool
	Events        []Event
}

// UpdatePlanSpecsInput is retained for callers performing a spec mutation.
type UpdatePlanSpecsInput struct {
	Workspace   string
	ProjectID   string
	PlanUID     int64
	Title       *string
	Description *string
	Deleted     *bool
	Specs       []*storepb.PlanConfig_Spec
}

// UpdatePlanSpecsResult is the result of a spec mutation.
type UpdatePlanSpecsResult = UpdatePlanResult

// UpdatePlanSpecs commits a Plan spec mutation through UpdatePlan.
func (w *Workflow) UpdatePlanSpecs(ctx context.Context, input UpdatePlanSpecsInput) (*UpdatePlanSpecsResult, error) {
	return w.UpdatePlan(ctx, UpdatePlanInput{
		Workspace:   input.Workspace,
		ProjectID:   input.ProjectID,
		PlanUID:     input.PlanUID,
		Title:       input.Title,
		Description: input.Description,
		Deleted:     input.Deleted,
		Specs:       &input.Specs,
	})
}

// UpdatePlan commits a Plan mutation and its linked review changes atomically.
func (w *Workflow) UpdatePlan(ctx context.Context, input UpdatePlanInput) (*UpdatePlanResult, error) {
	if w.beforePlanCommit != nil {
		w.beforePlanCommit()
	}
	tx, err := w.store.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to begin Plan review transaction")
	}
	defer tx.Rollback()

	if err := store.AcquirePlanIssueRolloutAdvisoryLock(ctx, tx, input.ProjectID, input.PlanUID); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to acquire Plan review lock")
	}
	issue, err := lockIssueByPlan(ctx, tx, input.ProjectID, input.PlanUID)
	if err != nil {
		return nil, err
	}
	plan, err := lockPlan(ctx, tx, input.Workspace, input.ProjectID, input.PlanUID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, workflowError(ErrorNotFound, "plan %d not found in project %s", input.PlanUID, input.ProjectID)
	}
	if input.Specs != nil && plan.Config.GetHasRollout() {
		return nil, workflowError(ErrorFailedPrecondition, "cannot update specs for plan that has a rollout")
	}

	oldSpecs := plan.Config.GetSpecs()
	updatedConfig := proto.CloneOf(plan.Config)
	if input.Specs != nil {
		updatedConfig.Specs = *input.Specs
		updatedConfig.ApprovalInputVersion = plan.Config.GetApprovalInputVersion() + 1
	}
	config, err := protojson.Marshal(updatedConfig)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to marshal Plan config")
	}

	updatedAt := time.Now()
	if err := tx.QueryRowContext(ctx, `
		UPDATE plan
		SET updated_at = $1,
			name = CASE WHEN $2 THEN $3 ELSE name END,
			description = CASE WHEN $4 THEN $5 ELSE description END,
			deleted = CASE WHEN $6 THEN $7 ELSE deleted END,
			config = CASE WHEN $8 THEN $9::jsonb ELSE config END
		WHERE project = $10
		  AND id = $11
		RETURNING updated_at`,
		updatedAt,
		input.Title != nil, stringValue(input.Title),
		input.Description != nil, stringValue(input.Description),
		input.Deleted != nil, boolValue(input.Deleted),
		input.Specs != nil, config,
		input.ProjectID, input.PlanUID,
	).Scan(&plan.UpdatedAt); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to update Plan")
	}
	if input.Title != nil {
		plan.Name = *input.Title
	}
	if input.Description != nil {
		plan.Description = *input.Description
	}
	if input.Deleted != nil {
		plan.Deleted = *input.Deleted
	}
	plan.Config = updatedConfig

	result := &UpdatePlanResult{Plan: plan, Issue: issue}
	if issue != nil && issue.Payload.GetDraft() {
		status := issue.Status
		if input.Deleted != nil {
			status = storepb.Issue_OPEN
			if *input.Deleted {
				status = storepb.Issue_CANCELED
			}
		}
		title := issue.Title
		if input.Title != nil {
			title = *input.Title
		}
		description := issue.Description
		if input.Description != nil {
			description = *input.Description
		}
		if err := tx.QueryRowContext(ctx, `
			UPDATE issue
			SET updated_at = $1,
				name = $2,
				description = $3,
				status = $4,
				ts_vector = $5
			WHERE project = $6
			  AND id = $7
			  AND COALESCE((payload->>'draft')::boolean, false)
			RETURNING updated_at`,
			time.Now(), title, description, status.String(), store.IssueSearchVector(title, description),
			input.ProjectID, issue.UID,
		).Scan(&issue.UpdatedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, workflowError(ErrorConflict, "linked draft issue was submitted while the Plan was being updated")
			}
			return nil, workflowWrap(ErrorInternal, err, "failed to synchronize linked draft issue")
		}
		issue.Title = title
		issue.Description = description
		issue.Status = status
	} else if issue != nil && input.Specs != nil {
		approval := &storepb.IssuePayloadApproval{
			ApprovalFindingDone:  false,
			ApprovalInputVersion: updatedConfig.GetApprovalInputVersion(),
		}
		if err := updateIssuePayload(ctx, tx, issue, &storepb.Issue{Approval: approval}, issuePayloadUpdateOptions{}); err != nil {
			return nil, workflowWrap(ErrorInternal, err, "failed to reset issue approval")
		}
		issue.Payload.Approval = approval
		result.ApprovalReset = true
	}
	if issue != nil && input.Specs != nil && !planSpecsEqualSet(oldSpecs, *input.Specs) {
		result.Events = append(result.Events, PlanUpdatedEvent{
			FromSpecs: oldSpecs,
			ToSpecs:   *input.Specs,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to commit Plan review transaction")
	}
	return result, nil
}

func lockPlan(ctx context.Context, tx *sql.Tx, workspace, projectID string, planUID int64) (*store.PlanMessage, error) {
	plan := &store.PlanMessage{Config: &storepb.PlanConfig{}}
	var config []byte
	err := tx.QueryRowContext(ctx, `
		SELECT plan.id, plan.creator, plan.created_at, plan.updated_at, plan.project, plan.name, plan.description, plan.config, plan.deleted
		FROM plan
		JOIN project ON project.resource_id = plan.project
		WHERE project.workspace = $1
		  AND plan.project = $2
		  AND plan.id = $3
		FOR UPDATE OF plan`, workspace, projectID, planUID).Scan(
		&plan.UID,
		&plan.Creator,
		&plan.CreatedAt,
		&plan.UpdatedAt,
		&plan.ProjectID,
		&plan.Name,
		&plan.Description,
		&config,
		&plan.Deleted,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to lock Plan")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(config, plan.Config); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to unmarshal Plan config")
	}
	return plan, nil
}

func planSpecsEqualSet(a, b []*storepb.PlanConfig_Spec) bool {
	if len(a) != len(b) {
		return false
	}
	byID := make(map[string]*storepb.PlanConfig_Spec, len(a))
	for _, spec := range a {
		byID[spec.GetId()] = spec
	}
	for _, spec := range b {
		other, ok := byID[spec.GetId()]
		if !ok || !proto.Equal(spec, other) {
			return false
		}
	}
	return true
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}
