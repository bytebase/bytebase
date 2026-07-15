package review

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// SubmittedEvent requests a submission timeline entry.
type SubmittedEvent struct{}

func (SubmittedEvent) isReviewEvent() {}

// IssueCreatedEvent requests the legacy issue-created webhook at submission.
type IssueCreatedEvent struct{}

func (IssueCreatedEvent) isReviewEvent() {}

// CreateDraftIssueInput describes a draft review issue linked to a Plan.
type CreateDraftIssueInput struct {
	Workspace string
	Issue     *store.IssueMessage
}

// CreateDraftIssueResult is the current linked draft and whether this call created it.
type CreateDraftIssueResult struct {
	Issue   *store.IssueMessage
	Created bool
}

// CreateDraftIssue creates a valid draft from current Plan state under the Plan lock.
func (w *Workflow) CreateDraftIssue(ctx context.Context, input CreateDraftIssueInput) (*CreateDraftIssueResult, error) {
	if input.Issue == nil || input.Issue.Type != storepb.Issue_DATABASE_CHANGE || input.Issue.PlanUID == nil || !input.Issue.Payload.GetDraft() {
		return nil, workflowError(ErrorInvalidAction, "draft review issue must have a database Plan")
	}
	if w.beforeCreateDraft != nil {
		w.beforeCreateDraft()
	}

	issue := *input.Issue
	issue.Payload = proto.CloneOf(input.Issue.Payload)
	issue.Payload.Labels = store.CanonicalizeIssueLabels(issue.Payload.GetLabels())
	issue.Status = storepb.Issue_OPEN

	tx, err := w.store.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to begin draft creation transaction")
	}
	defer tx.Rollback()
	key := issue.ProjectID + "/" + strconv.FormatInt(*issue.PlanUID, 10)
	if err := store.AcquireAdvisoryXactLockWithStringKey(ctx, tx, store.AdvisoryLockKeyPlanIssueRollout, key); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to acquire Plan review lock")
	}
	existing, err := lockIssueByPlan(ctx, tx, issue.ProjectID, *issue.PlanUID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.Payload.GetDraft() && existing.CreatorEmail == issue.CreatorEmail {
			return &CreateDraftIssueResult{Issue: existing}, nil
		}
		return nil, workflowError(ErrorConflict, "Plan already has a review issue")
	}
	plan, err := lockPlan(ctx, tx, input.Workspace, issue.ProjectID, *issue.PlanUID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, workflowError(ErrorNotFound, "Plan not found")
	}
	if plan.Deleted {
		return nil, workflowError(ErrorFailedPrecondition, "cannot create a draft issue for a closed Plan")
	}
	if plan.Config.GetHasRollout() {
		return nil, workflowError(ErrorFailedPrecondition, "cannot create a draft issue because the Plan already has a rollout")
	}
	if _, err := classifyReviewPlan(plan); err != nil {
		return nil, err
	}
	issue.Title = plan.Name
	issue.Description = plan.Description
	if _, err := tx.ExecContext(ctx, `
		SELECT 1 FROM project
		WHERE workspace = $1 AND resource_id = $2
		FOR UPDATE`, input.Workspace, issue.ProjectID); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to lock project for draft creation")
	}
	if err := tx.QueryRowContext(ctx, `
		SELECT GREATEST(COALESCE(MAX(id), 0) + 1, 101)
		FROM issue
		WHERE project = $1`, issue.ProjectID).Scan(&issue.UID); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to allocate issue ID")
	}
	payload, err := protojson.Marshal(issue.Payload)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to marshal draft issue payload")
	}
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO issue (
			id, creator, project, plan_id, name, status, type, description, payload, ts_vector
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`,
		issue.UID, issue.CreatorEmail, issue.ProjectID, issue.PlanUID, issue.Title,
		issue.Status.String(), issue.Type.String(), issue.Description, payload,
		store.IssueSearchVector(issue.Title, issue.Description),
	).Scan(&issue.CreatedAt, &issue.UpdatedAt); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to insert draft issue")
	}
	if err := tx.Commit(); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to commit draft creation")
	}
	return &CreateDraftIssueResult{Issue: &issue, Created: true}, nil
}

// SubmitIssueInput identifies a draft issue submission.
type SubmitIssueInput struct {
	Workspace string
	ProjectID string
	IssueUID  int64
	Labels    []string
	LabelsSet bool
}

// SubmitIssueResult is the committed issue and its post-commit effects.
type SubmitIssueResult struct {
	Issue          *store.IssueMessage
	Project        *store.ProjectMessage
	Submitted      bool
	LabelsChanged  bool
	PreviousLabels []string
	Events         []Event
}

// SubmitIssue validates and atomically submits a draft review issue.
func (w *Workflow) SubmitIssue(ctx context.Context, input SubmitIssueInput) (*SubmitIssueResult, error) {
	project, err := w.store.GetProject(ctx, &store.FindProjectMessage{
		Workspace: input.Workspace, ResourceID: &input.ProjectID,
	})
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to get project")
	}
	if project == nil {
		return nil, workflowError(ErrorNotFound, "project %s not found", input.ProjectID)
	}
	observedIssue, err := w.store.GetIssue(ctx, &store.FindIssueMessage{
		Workspace: input.Workspace, ProjectIDs: []string{input.ProjectID}, UID: &input.IssueUID,
	})
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to get issue")
	}
	if observedIssue == nil {
		return nil, workflowError(ErrorNotFound, "issue %d not found in project %s", input.IssueUID, input.ProjectID)
	}
	if !observedIssue.Payload.GetDraft() {
		return &SubmitIssueResult{Issue: observedIssue, Project: project}, nil
	}
	if observedIssue.Type != storepb.Issue_DATABASE_CHANGE || observedIssue.PlanUID == nil {
		return nil, workflowError(ErrorFailedPrecondition, "draft review issue must have a database Plan")
	}
	if w.beforeSubmit != nil {
		w.beforeSubmit()
	}

	tx, err := w.store.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to begin issue submission transaction")
	}
	defer tx.Rollback()

	key := input.ProjectID + "/" + strconv.FormatInt(*observedIssue.PlanUID, 10)
	if err := store.AcquireAdvisoryXactLockWithStringKey(ctx, tx, store.AdvisoryLockKeyPlanIssueRollout, key); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to acquire Plan review lock")
	}
	issue, err := lockIssue(ctx, tx, input.ProjectID, input.IssueUID)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, workflowError(ErrorNotFound, "issue %d not found in project %s", input.IssueUID, input.ProjectID)
	}
	if !issue.Payload.GetDraft() {
		return &SubmitIssueResult{Issue: issue, Project: project}, nil
	}
	if issue.Type != storepb.Issue_DATABASE_CHANGE || issue.PlanUID == nil || *issue.PlanUID != *observedIssue.PlanUID {
		return nil, workflowError(ErrorFailedPrecondition, "draft review issue must have a database Plan")
	}
	plan, err := lockIssuePlan(ctx, tx, issue)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, workflowError(ErrorNotFound, "plan not found")
	}

	labels := store.CanonicalizeIssueLabels(issue.Payload.GetLabels())
	previousLabels := labels
	if input.LabelsSet {
		labels = store.CanonicalizeIssueLabels(input.Labels)
	}
	if err := validateSubmissionState(project, issue, plan, labels); err != nil {
		return nil, err
	}
	if reviewPlanRequiresChecks(plan) {
		planCheckRun, err := lockPlanCheckRun(ctx, tx, input.ProjectID, plan.UID)
		if err != nil {
			return nil, err
		}
		if err := validatePlanCheckRun(project, plan, planCheckRun); err != nil {
			return nil, err
		}
	}

	if err := submitIssuePayload(ctx, tx, issue, labels); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to submit issue")
	}
	issue.Payload.Draft = false
	issue.Payload.Labels = labels
	if err := tx.Commit(); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to commit issue submission")
	}
	return &SubmitIssueResult{
		Issue:          issue,
		Project:        project,
		Submitted:      true,
		LabelsChanged:  !slices.Equal(previousLabels, labels),
		PreviousLabels: previousLabels,
		Events:         []Event{SubmittedEvent{}, IssueCreatedEvent{}, ApprovalCheckEvent{}},
	}, nil
}

type reviewPlanKind int

const (
	reviewPlanCreateDatabase reviewPlanKind = iota
	reviewPlanChangeDatabase
)

func validateSubmissionState(project *store.ProjectMessage, issue *store.IssueMessage, plan *store.PlanMessage, labels []string) error {
	if issue.Status != storepb.Issue_OPEN {
		return workflowError(ErrorConflict, "draft issue status changed while it was being submitted")
	}
	if strings.TrimSpace(issue.Title) == "" || strings.TrimSpace(plan.Name) == "" {
		return workflowError(ErrorInvalidAction, "issue and Plan title are required")
	}
	if project.Setting.GetForceIssueLabels() && len(labels) == 0 {
		return workflowError(ErrorInvalidAction, "require issue labels")
	}
	if plan.Deleted {
		return workflowError(ErrorFailedPrecondition, "cannot submit an issue for a closed Plan")
	}
	if plan.Config.GetHasRollout() {
		return workflowError(ErrorFailedPrecondition, "cannot submit an issue after rollout has started")
	}
	kind, err := classifyReviewPlan(plan)
	if err != nil {
		return err
	}
	for index, spec := range plan.Config.GetSpecs() {
		if err := validateReviewPlanSpecReady(spec, index, kind); err != nil {
			return err
		}
	}
	return nil
}

func classifyReviewPlan(plan *store.PlanMessage) (reviewPlanKind, error) {
	var kind reviewPlanKind
	for index, spec := range plan.Config.GetSpecs() {
		if spec == nil {
			return 0, workflowError(ErrorInvalidAction, "draft issues require a database Plan")
		}
		var current reviewPlanKind
		switch config := spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			current = reviewPlanCreateDatabase
		case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
			if config.ChangeDatabaseConfig.GetRelease() != "" {
				return 0, workflowError(ErrorInvalidAction, "draft issues are not supported for GitOps Plans")
			}
			current = reviewPlanChangeDatabase
		default:
			return 0, workflowError(ErrorInvalidAction, "draft issues require a database Plan")
		}
		if index > 0 && current != kind {
			return 0, workflowError(ErrorInvalidAction, "draft issues are not supported for mixed Plans")
		}
		kind = current
	}
	if len(plan.Config.GetSpecs()) == 0 {
		return 0, workflowError(ErrorInvalidAction, "draft issues require a database Plan")
	}
	return kind, nil
}

func validateReviewPlanSpecReady(spec *storepb.PlanConfig_Spec, index int, kind reviewPlanKind) error {
	reference := fmt.Sprintf("Plan spec at index %d", index)
	if id := spec.GetId(); id != "" {
		reference = fmt.Sprintf("Plan spec %q", id)
	}
	switch kind {
	case reviewPlanCreateDatabase:
		config := spec.GetCreateDatabaseConfig()
		if strings.TrimSpace(config.GetTarget()) == "" {
			return workflowError(ErrorInvalidAction, "%s is missing create target", reference)
		}
		if strings.TrimSpace(config.GetDatabase()) == "" {
			return workflowError(ErrorInvalidAction, "%s is missing create database name", reference)
		}
	case reviewPlanChangeDatabase:
		config := spec.GetChangeDatabaseConfig()
		if len(config.GetTargets()) == 0 {
			return workflowError(ErrorInvalidAction, "%s is missing change targets", reference)
		}
		if config.GetSheetSha256() == "" {
			return workflowError(ErrorInvalidAction, "%s is missing sheet", reference)
		}
	default:
		return workflowError(ErrorInternal, "unsupported review Plan kind %d", kind)
	}
	return nil
}

func reviewPlanRequiresChecks(plan *store.PlanMessage) bool {
	for _, spec := range plan.Config.GetSpecs() {
		config := spec.GetChangeDatabaseConfig()
		if config != nil && config.GetRelease() == "" && len(config.GetTargets()) > 0 {
			return true
		}
	}
	return false
}

func lockPlanCheckRun(ctx context.Context, tx *sql.Tx, projectID string, planUID int64) (*store.PlanCheckRunMessage, error) {
	run := &store.PlanCheckRunMessage{Result: &storepb.PlanCheckRunResult{}}
	var result []byte
	err := tx.QueryRowContext(ctx, `
		SELECT id, created_at, updated_at, project, plan_id, status, result
		FROM plan_check_run
		WHERE project = $1 AND plan_id = $2
		FOR UPDATE`, projectID, planUID).Scan(
		&run.UID, &run.CreatedAt, &run.UpdatedAt, &run.ProjectID, &run.PlanUID, &run.Status, &result,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to lock Plan check run")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(result, run.Result); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to unmarshal Plan check result")
	}
	return run, nil
}

func validatePlanCheckRun(project *store.ProjectMessage, plan *store.PlanMessage, run *store.PlanCheckRunMessage) error {
	if run == nil {
		return workflowError(ErrorFailedPrecondition, "Plan checks have not run")
	}
	if run.Result.GetApprovalInputVersion() != plan.Config.GetApprovalInputVersion() {
		return workflowError(ErrorFailedPrecondition, "Plan checks are stale")
	}
	switch run.Status {
	case store.PlanCheckRunStatusAvailable, store.PlanCheckRunStatusRunning:
		return workflowError(ErrorFailedPrecondition, "Plan checks are still running")
	case store.PlanCheckRunStatusCanceled, store.PlanCheckRunStatusFailed:
		return workflowError(ErrorFailedPrecondition, "Plan checks did not pass")
	case store.PlanCheckRunStatusDone:
	default:
		return workflowError(ErrorFailedPrecondition, "Plan checks are not complete")
	}
	for _, result := range run.Result.GetResults() {
		if result.GetStatus() != storepb.Advice_ERROR {
			continue
		}
		if project.Setting.GetRequirePlanCheckNoError() ||
			(project.Setting.GetEnforceSqlReview() && result.GetType() == storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE) {
			return workflowError(ErrorFailedPrecondition, "Plan checks did not pass")
		}
	}
	return nil
}

func submitIssuePayload(ctx context.Context, tx *sql.Tx, issue *store.IssueMessage, labels []string) error {
	patch := &storepb.Issue{Labels: labels}
	return updateIssuePayload(ctx, tx, issue, patch, issuePayloadUpdateOptions{
		removeLabels: len(labels) == 0,
		submitDraft:  true,
	})
}
