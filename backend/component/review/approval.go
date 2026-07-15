package review

import (
	"context"
	"slices"

	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// ApprovalEvaluator owns license-dependent approval evaluation and its
// transactional commit through the review workflow.
type ApprovalEvaluator struct {
	workflow         *Workflow
	licenseService   *enterprise.LicenseService
	evaluateApproval func(context.Context, *store.IssueMessage, *store.ProjectMessage, *storepb.WorkspaceApprovalSetting) error
	approvalSetting  *storepb.WorkspaceApprovalSetting
	beforeCommit     func()
}

// NewApprovalEvaluator creates an approval evaluator with all production dependencies.
func NewApprovalEvaluator(store *store.Store, licenseService *enterprise.LicenseService) *ApprovalEvaluator {
	return &ApprovalEvaluator{
		workflow:       NewWorkflow(store),
		licenseService: licenseService,
	}
}

// ApplyApprovalTemplateInput identifies a Bytebase Issue awaiting approval finding.
type ApplyApprovalTemplateInput struct {
	Workspace string
	ProjectID string
	IssueUID  int64
}

// ApplyApprovalTemplateResult is the committed approval finding and its effects.
type ApplyApprovalTemplateResult struct {
	Issue   *store.IssueMessage
	Project *store.ProjectMessage
	Applied bool
	Events  []Event
}

// ApplyApprovalTemplate computes and commits an approval finding while keeping
// snapshot validation private to the module.
func (a *ApprovalEvaluator) ApplyApprovalTemplate(ctx context.Context, input ApplyApprovalTemplateInput) (*ApplyApprovalTemplateResult, error) {
	w := a.workflow
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
	result := &ApplyApprovalTemplateResult{Issue: issue, Project: project}
	if issue.Payload.GetDraft() {
		return result, nil
	}

	var observedPlan *store.PlanMessage
	var approvalInputVersion int64
	if issue.Type == storepb.Issue_DATABASE_CHANGE {
		if issue.PlanUID == nil {
			return nil, workflowError(ErrorFailedPrecondition, "expected plan UID in issue %d", issue.UID)
		}
		observedPlan, err = w.store.GetPlan(ctx, &store.FindPlanMessage{
			Workspace: input.Workspace,
			ProjectID: input.ProjectID,
			UID:       issue.PlanUID,
		})
		if err != nil {
			return nil, workflowWrap(ErrorInternal, err, "failed to get plan")
		}
		if observedPlan == nil {
			return nil, workflowError(ErrorNotFound, "plan %d not found", *issue.PlanUID)
		}
		approvalInputVersion = observedPlan.Config.GetApprovalInputVersion()
	}
	observedApproval := issue.Payload.GetApproval()
	if observedApproval != nil && observedApproval.GetApprovalFindingDone() && observedApproval.GetApprovalInputVersion() == approvalInputVersion {
		return result, nil
	}
	observedLabels := store.CanonicalizeIssueLabels(issue.Payload.GetLabels())
	evaluatedIssue := *issue
	evaluatedIssue.Payload = proto.CloneOf(issue.Payload)
	evaluate := a.evaluateApproval
	approvalSetting := a.approvalSetting
	if evaluate == nil {
		if a.licenseService == nil {
			return nil, workflowError(ErrorInternal, "approval evaluation is not configured")
		}
		if approvalSetting == nil {
			approvalSetting, err = w.store.GetWorkspaceApprovalSetting(ctx, project.Workspace)
			if err != nil {
				return nil, workflowWrap(ErrorInternal, err, "failed to get workspace approval setting")
			}
		}
		evaluate = func(ctx context.Context, issue *store.IssueMessage, project *store.ProjectMessage, setting *storepb.WorkspaceApprovalSetting) error {
			return evaluateApprovalTemplateForIssue(ctx, w.store, a.licenseService, issue, project, setting)
		}
	}
	if approvalSetting == nil {
		approvalSetting = &storepb.WorkspaceApprovalSetting{}
	}
	if err := evaluate(ctx, &evaluatedIssue, project, approvalSetting); err != nil {
		return nil, err
	}
	evaluatedApproval := evaluatedIssue.Payload.GetApproval()
	evaluatedRiskLevel := evaluatedIssue.Payload.GetRiskLevel()
	if approvalsEqual(evaluatedApproval, observedApproval) && evaluatedRiskLevel == issue.Payload.GetRiskLevel() {
		return result, nil
	}
	if evaluatedApproval == nil || evaluatedApproval.GetApprovalInputVersion() != approvalInputVersion {
		return nil, workflowError(ErrorConflict, "approval finding used stale input")
	}
	if a.beforeCommit != nil {
		a.beforeCommit()
	}

	tx, err := w.store.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to begin approval finding transaction")
	}
	defer tx.Rollback()
	lockedIssue, err := lockIssue(ctx, tx, input.ProjectID, input.IssueUID)
	if err != nil {
		return nil, err
	}
	if lockedIssue == nil {
		return nil, workflowError(ErrorNotFound, "issue %d not found in project %s", input.IssueUID, input.ProjectID)
	}
	lockedPlan, err := lockIssuePlan(ctx, tx, lockedIssue)
	if err != nil {
		return nil, err
	}
	if !approvalsEqual(lockedIssue.Payload.GetApproval(), observedApproval) ||
		!slices.Equal(store.CanonicalizeIssueLabels(lockedIssue.Payload.GetLabels()), observedLabels) ||
		lockedIssue.Payload.GetDraft() != issue.Payload.GetDraft() ||
		lockedIssue.Type != issue.Type ||
		!sameInt64Pointer(lockedIssue.PlanUID, issue.PlanUID) {
		return nil, workflowError(ErrorConflict, "approval finding input changed")
	}
	if observedPlan != nil && (lockedPlan == nil || lockedPlan.Config.GetApprovalInputVersion() != approvalInputVersion) {
		return nil, workflowError(ErrorConflict, "approval finding input changed")
	}
	if lockedPlan != nil && project.Setting.GetRequireIssueApproval() && lockedPlan.Config.GetHasRollout() {
		return nil, workflowError(ErrorConflict, "rollout already started")
	}

	if err := updateIssuePayload(ctx, tx, lockedIssue, &storepb.Issue{
		Approval:  evaluatedApproval,
		RiskLevel: evaluatedRiskLevel,
	}, false); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to apply approval finding")
	}
	lockedIssue.Payload.Approval = evaluatedApproval
	lockedIssue.Payload.RiskLevel = evaluatedRiskLevel
	if err := tx.Commit(); err != nil {
		return nil, workflowWrap(ErrorInternal, err, "failed to commit approval finding transaction")
	}
	result.Issue = lockedIssue
	result.Applied = true
	result.Events = []Event{ApprovalRequestedEvent{}}
	return result, nil
}

func approvalsEqual(a, b *storepb.IssuePayloadApproval) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}
