// Package review coordinates Bytebase Issue and Plan review transitions.
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
	"github.com/bytebase/bytebase/backend/utils"
)

// ErrorCode classifies a rejected review transition.
type ErrorCode int

const (
	ErrorInternal ErrorCode = iota
	ErrorNotFound
	ErrorInvalidAction
	ErrorFailedPrecondition
	ErrorPermissionDenied
	ErrorConflict
)

// ErrorReason identifies a domain-specific reason within an error class.
type ErrorReason int

const (
	ReasonUnspecified ErrorReason = iota
	ReasonDraftIssue
	ReasonApprovalRequired
	ReasonStaleInput
)

// Error is a typed review transition error.
type Error struct {
	Code   ErrorCode
	Reason ErrorReason
	Err    error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// Action is an interactive Bytebase Issue review action.
type Action int

const (
	ActionApprove Action = iota
	ActionReject
	ActionRequest
)

// Event describes a post-commit review effect.
type Event interface {
	isReviewEvent()
}

// IssueCommentEvent requests an approval timeline entry.
type IssueCommentEvent struct {
	ActorEmail     string
	Comment        string
	ApprovalStatus storepb.IssuePayloadApproval_Approver_Status
}

func (IssueCommentEvent) isReviewEvent() {}

// ApprovalRequestedEvent requests an approval notification.
type ApprovalRequestedEvent struct{}

func (ApprovalRequestedEvent) isReviewEvent() {}

// IssueApprovedEvent requests an approved notification.
type IssueApprovedEvent struct{}

func (IssueApprovedEvent) isReviewEvent() {}

// IssueSentBackEvent requests a sent-back notification.
type IssueSentBackEvent struct{}

func (IssueSentBackEvent) isReviewEvent() {}

// CompleteAccessRequestEvent requests access or role grant completion.
type CompleteAccessRequestEvent struct{}

func (CompleteAccessRequestEvent) isReviewEvent() {}

// CreateRolloutEvent requests rollout creation.
type CreateRolloutEvent struct{}

func (CreateRolloutEvent) isReviewEvent() {}

// PlanUpdatedEvent requests a Plan spec audit entry.
type PlanUpdatedEvent struct {
	FromSpecs []*storepb.PlanConfig_Spec
	ToSpecs   []*storepb.PlanConfig_Spec
}

func (PlanUpdatedEvent) isReviewEvent() {}

// ApprovalCheckEvent requests approval reevaluation.
type ApprovalCheckEvent struct{}

func (ApprovalCheckEvent) isReviewEvent() {}

// CompleteRolloutIssueEvent requests linked Bytebase Issue completion.
type CompleteRolloutIssueEvent struct{}

func (CompleteRolloutIssueEvent) isReviewEvent() {}

// IssueInput identifies an interactive review transition.
type IssueInput struct {
	Workspace string
	ProjectID string
	IssueUID  int64
	Actor     *store.UserMessage
	Action    Action
	Comment   string
}

// IssueResult is the committed state and its post-commit effects.
type IssueResult struct {
	Issue    *store.IssueMessage
	Project  *store.ProjectMessage
	Approved bool
	Events   []Event
}

// Workflow owns transactional Bytebase Issue and Plan review transitions.
type Workflow struct {
	store            *store.Store
	beforeCommit     func()
	beforePlanCommit func()
}

// NewWorkflow creates a review workflow.
func NewWorkflow(store *store.Store) *Workflow {
	return &Workflow{store: store}
}

// ReviewIssue applies an interactive approval action atomically.
func (w *Workflow) ReviewIssue(ctx context.Context, input IssueInput) (*IssueResult, error) {
	if input.Actor == nil {
		return nil, workflowError(ErrorInternal, "user not found")
	}
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
	issue, err := w.store.GetIssue(ctx, &store.FindIssueMessage{
		Workspace:  input.Workspace,
		ProjectIDs: []string{input.ProjectID},
		UID:        &input.IssueUID,
	})
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to get issue")
	}
	if issue == nil {
		return nil, workflowError(ErrorNotFound, "issue %d not found in project %s", input.IssueUID, input.ProjectID)
	}
	if issue.Payload.GetDraft() {
		return nil, workflowError(ErrorFailedPrecondition, "draft issue must be submitted before approval actions are allowed")
	}
	approval := issue.Payload.GetApproval()
	if approval == nil {
		return nil, workflowError(ErrorInternal, "issue payload approval is nil")
	}
	if !approval.GetApprovalFindingDone() {
		return nil, workflowError(ErrorFailedPrecondition, "approval template finding is not done")
	}
	if approval.GetApprovalTemplate() == nil {
		return nil, workflowError(ErrorInternal, "approval template is required")
	}

	var observedPlan *store.PlanMessage
	if issue.Type == storepb.Issue_DATABASE_CHANGE && issue.PlanUID != nil {
		observedPlan, err = w.store.GetPlan(ctx, &store.FindPlanMessage{
			Workspace: input.Workspace,
			ProjectID: input.ProjectID,
			UID:       issue.PlanUID,
		})
		if err != nil {
			return nil, workflowWrap(ErrorInternal, err, "failed to get plan")
		}
		if observedPlan == nil {
			return nil, workflowError(ErrorNotFound, "plan not found")
		}
		if approval.GetApprovalInputVersion() != observedPlan.Config.GetApprovalInputVersion() {
			return nil, workflowError(ErrorConflict, "approval finding is stale")
		}
	}

	updatedApproval := proto.CloneOf(approval)
	events, err := w.applyReviewAction(ctx, project, issue, input, updatedApproval)
	if err != nil {
		return nil, err
	}
	if w.beforeCommit != nil {
		w.beforeCommit()
	}

	tx, err := w.store.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to begin review transaction")
	}
	defer tx.Rollback()

	lockedIssue, err := lockIssue(ctx, tx, input.ProjectID, input.IssueUID)
	if err != nil {
		return nil, err
	}
	if lockedIssue == nil {
		return nil, workflowError(ErrorNotFound, "issue %d not found in project %s", input.IssueUID, input.ProjectID)
	}
	plan, err := lockIssuePlan(ctx, tx, lockedIssue)
	if err != nil {
		return nil, err
	}
	lockedApproval := lockedIssue.Payload.GetApproval()
	if lockedIssue.Payload.GetDraft() != issue.Payload.GetDraft() ||
		lockedIssue.Type != issue.Type ||
		!sameInt64Pointer(lockedIssue.PlanUID, issue.PlanUID) ||
		lockedApproval == nil || !lockedApproval.Equal(approval) {
		return nil, workflowError(ErrorConflict, "approval finding is stale")
	}
	if observedPlan != nil && (plan == nil || plan.Config.GetApprovalInputVersion() != observedPlan.Config.GetApprovalInputVersion()) {
		return nil, workflowError(ErrorConflict, "approval finding is stale")
	}

	if err := updateIssuePayload(ctx, tx, lockedIssue, &storepb.Issue{Approval: updatedApproval}, false); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to update issue approval")
	}
	lockedIssue.Payload.Approval = updatedApproval

	approved, err := utils.CheckIssueApprovedForPlan(lockedIssue, plan)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to check if the issue is approved")
	}
	if err := tx.Commit(); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to commit review transaction")
	}
	if approved && input.Action == ActionApprove {
		events = append(events, IssueApprovedEvent{})
		switch lockedIssue.Type {
		case storepb.Issue_ACCESS_GRANT, storepb.Issue_ROLE_GRANT:
			events = append(events, CompleteAccessRequestEvent{})
		case storepb.Issue_DATABASE_CHANGE:
			if lockedIssue.PlanUID != nil {
				events = append(events, CreateRolloutEvent{})
			}
		default:
		}
	}

	return &IssueResult{
		Issue:    lockedIssue,
		Project:  project,
		Approved: approved,
		Events:   events,
	}, nil
}

func updateIssuePayload(ctx context.Context, tx *sql.Tx, issue *store.IssueMessage, patch *storepb.Issue, removeLabels bool) error {
	payload, err := protojson.Marshal(patch)
	if err != nil {
		return errors.Wrap(err, "failed to marshal issue payload")
	}
	return tx.QueryRowContext(ctx, `
		UPDATE issue
		SET updated_at = $1,
			payload = payload || $2::jsonb || CASE WHEN $5 THEN jsonb_build_object('labels', NULL) ELSE '{}'::jsonb END
		WHERE project = $3
		  AND id = $4
		RETURNING updated_at`,
		time.Now(), payload, issue.ProjectID, issue.UID, removeLabels,
	).Scan(&issue.UpdatedAt)
}

func (w *Workflow) applyReviewAction(ctx context.Context, project *store.ProjectMessage, issue *store.IssueMessage, input IssueInput, approval *storepb.IssuePayloadApproval) ([]Event, error) {
	switch input.Action {
	case ActionApprove, ActionReject:
		verb := "approve"
		status := storepb.IssuePayloadApproval_Approver_APPROVED
		if input.Action == ActionReject {
			verb = "reject"
			status = storepb.IssuePayloadApproval_Approver_REJECTED
		}
		if utils.FindRejectedRole(approval) != "" {
			return nil, workflowError(ErrorInvalidAction, "cannot %s because the issue has been rejected", verb)
		}
		role := utils.FindNextPendingRole(approval)
		if role == "" {
			return nil, workflowError(ErrorInvalidAction, "the issue has been approved")
		}
		canReview, err := w.canReview(ctx, project, input.Actor, role)
		if err != nil {
			return nil, err
		}
		if !canReview {
			return nil, workflowError(ErrorPermissionDenied, "cannot %s because the user does not have the required permission", verb)
		}
		if !project.Setting.GetAllowSelfApproval() && issue.CreatorEmail == input.Actor.Email {
			return nil, workflowError(ErrorPermissionDenied, "cannot %s because self-approval is not allowed for this project", verb)
		}
		approval.Approvers = append(approval.Approvers, &storepb.IssuePayloadApproval_Approver{
			Status:    status,
			Principal: common.FormatUserEmail(input.Actor.Email),
		})
		events := []Event{IssueCommentEvent{
			ActorEmail:     input.Actor.Email,
			Comment:        input.Comment,
			ApprovalStatus: status,
		}}
		if input.Action == ActionApprove {
			events = append(events, ApprovalRequestedEvent{})
		} else {
			events = append(events, IssueSentBackEvent{})
		}
		return events, nil
	case ActionRequest:
		if utils.FindRejectedRole(approval) == "" {
			return nil, workflowError(ErrorInvalidAction, "cannot request issues because the issue is not rejected")
		}
		if issue.CreatorEmail != input.Actor.Email {
			return nil, workflowError(ErrorPermissionDenied, "cannot request issues because you are not the issue creator")
		}

		approvers := approval.Approvers[:0]
		for _, approver := range approval.Approvers {
			if approver.Status != storepb.IssuePayloadApproval_Approver_REJECTED {
				approvers = append(approvers, approver)
			}
		}
		approval.Approvers = approvers
		return []Event{
			ApprovalRequestedEvent{},
			IssueCommentEvent{
				ActorEmail:     input.Actor.Email,
				Comment:        input.Comment,
				ApprovalStatus: storepb.IssuePayloadApproval_Approver_PENDING,
			},
		}, nil
	default:
		return nil, workflowError(ErrorInvalidAction, "unsupported review action")
	}
}

func (w *Workflow) canReview(ctx context.Context, project *store.ProjectMessage, user *store.UserMessage, role string) (bool, error) {
	projectPolicy, err := w.store.GetProjectIamPolicy(ctx, project.Workspace, project.ResourceID)
	if err != nil {
		return false, workflowWrap(ErrorInternal, err, "failed to get project IAM policy")
	}
	workspacePolicy, err := w.store.GetWorkspaceIamPolicy(ctx, project.Workspace)
	if err != nil {
		return false, workflowWrap(ErrorInternal, err, "failed to get workspace IAM policy")
	}
	roles := utils.GetUserFormattedRolesMap(ctx, w.store, project.Workspace, user, projectPolicy.Policy, workspacePolicy.Policy)
	return roles[role], nil
}

func lockIssue(ctx context.Context, tx *sql.Tx, projectID string, issueUID int64) (*store.IssueMessage, error) {
	return scanLockedIssue(tx.QueryRowContext(ctx, `
		SELECT id, creator, created_at, updated_at, project, plan_id, name, status, type, description, payload
		FROM issue
		WHERE project = $1
		  AND id = $2
		FOR UPDATE`, projectID, issueUID))
}

func lockIssueByPlan(ctx context.Context, tx *sql.Tx, projectID string, planUID int64) (*store.IssueMessage, error) {
	return scanLockedIssue(tx.QueryRowContext(ctx, `
		SELECT id, creator, created_at, updated_at, project, plan_id, name, status, type, description, payload
		FROM issue
		WHERE project = $1
		  AND plan_id = $2
		FOR UPDATE`, projectID, planUID))
}

func scanLockedIssue(row *sql.Row) (*store.IssueMessage, error) {
	issue := &store.IssueMessage{Payload: &storepb.Issue{}}
	var payload []byte
	var status string
	var issueType string
	err := row.Scan(
		&issue.UID,
		&issue.CreatorEmail,
		&issue.CreatedAt,
		&issue.UpdatedAt,
		&issue.ProjectID,
		&issue.PlanUID,
		&issue.Title,
		&status,
		&issueType,
		&issue.Description,
		&payload,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to lock issue")
	}
	statusValue, ok := storepb.Issue_Status_value[status]
	if !ok {
		return nil, workflowError(ErrorInternal, "invalid issue status %q", status)
	}
	issue.Status = storepb.Issue_Status(statusValue)
	typeValue, ok := storepb.Issue_Type_value[issueType]
	if !ok {
		return nil, workflowError(ErrorInternal, "invalid issue type %q", issueType)
	}
	issue.Type = storepb.Issue_Type(typeValue)
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, issue.Payload); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to unmarshal issue payload")
	}
	return issue, nil
}

func lockIssuePlan(ctx context.Context, tx *sql.Tx, issue *store.IssueMessage) (*store.PlanMessage, error) {
	if issue.Type != storepb.Issue_DATABASE_CHANGE || issue.PlanUID == nil {
		return nil, nil
	}
	plan := &store.PlanMessage{Config: &storepb.PlanConfig{}}
	var config []byte
	err := tx.QueryRowContext(ctx, `
		SELECT id, creator, created_at, updated_at, project, name, description, config, deleted
		FROM plan
		WHERE project = $1
		  AND id = $2
		FOR UPDATE`, issue.ProjectID, *issue.PlanUID).Scan(
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
		return nil, workflowError(ErrorNotFound, "plan not found")
	}
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to lock plan")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(config, plan.Config); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to unmarshal plan config")
	}
	return plan, nil
}

func workflowError(code ErrorCode, format string, args ...any) error {
	return &Error{Code: code, Err: errors.Errorf(format, args...)}
}

func workflowReasonError(code ErrorCode, reason ErrorReason, message string) error {
	return &Error{Code: code, Reason: reason, Err: errors.New(message)}
}

func workflowWrap(_ ErrorCode, err error, message string) error {
	return &Error{Code: ErrorInternal, Err: errors.Wrap(err, message)}
}

func sameInt64Pointer(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
