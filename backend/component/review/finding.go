package review

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func findApprovalTemplateForIssue(ctx context.Context, stores *store.Store, licenseService *enterprise.LicenseService, issue *store.IssueMessage, settingForTest ...*storepb.WorkspaceApprovalSetting) (*ApplyApprovalTemplateResult, error) {
	project, err := stores.GetProjectByResourceID(ctx, issue.ProjectID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project")
	}
	if project == nil {
		return nil, errors.Errorf("project %s not found", issue.ProjectID)
	}

	evaluator := NewApprovalEvaluator(stores, licenseService)
	if len(settingForTest) > 0 {
		evaluator.approvalSetting = settingForTest[0]
	}
	result, err := evaluator.ApplyApprovalTemplate(ctx, ApplyApprovalTemplateInput{
		Workspace: project.Workspace,
		ProjectID: issue.ProjectID,
		IssueUID:  issue.UID,
	})
	if err != nil {
		var workflowErr *Error
		if errors.As(err, &workflowErr) && workflowErr.Code == ErrorConflict {
			return &ApplyApprovalTemplateResult{Issue: issue, Project: project}, nil
		}
		return nil, err
	}
	if !result.Applied {
		return result, nil
	}
	issue.Payload = result.Issue.Payload
	return result, nil
}

// DispatchApprovalEvents delivers post-commit effects from approval evaluation.
func DispatchApprovalEvents(ctx context.Context, stores *store.Store, webhookManager *webhook.Manager, result *ApplyApprovalTemplateResult) {
	if result == nil || result.Issue == nil || result.Project == nil {
		return
	}
	for _, event := range result.Events {
		if _, ok := event.(ApprovalRequestedEvent); ok {
			NotifyApprovalRequested(ctx, stores, webhookManager, result.Issue, result.Project)
		}
	}
}

func evaluateApprovalTemplateForIssue(
	ctx context.Context,
	stores *store.Store,
	licenseService *enterprise.LicenseService,
	issue *store.IssueMessage,
	project *store.ProjectMessage,
	approvalSetting *storepb.WorkspaceApprovalSetting,
) error {
	if issue.Payload.GetDraft() {
		return nil
	}
	approvalInputVersion, err := getIssuePlanApprovalInputVersion(ctx, stores, issue)
	if err != nil {
		return err
	}
	approvalLabels := store.CanonicalizeIssueLabels(issue.Payload.GetLabels())
	if approvalLabels == nil {
		approvalLabels = []string{}
	}

	approvalTemplate, celVarsList, approvalInputVersion, done, err := func() (*storepb.ApprovalTemplate, []map[string]any, int64, bool, error) {
		if licenseService.IsFeatureEnabled(ctx, project.Workspace, v1pb.PlanFeature_FEATURE_APPROVAL_WORKFLOW) != nil {
			// An unavailable approval-workflow feature intentionally falls back to no approval template.
			return nil, nil, approvalInputVersion, true, nil //nolint:nilerr
		}

		approvalSource, err := getApprovalSourceFromIssue(ctx, stores, issue)
		if err != nil {
			return nil, nil, approvalInputVersion, false, errors.Wrap(err, "failed to get approval source from issue")
		}
		if approvalSource == storepb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED {
			return nil, nil, approvalInputVersion, true, nil
		}

		celVarsList, celApprovalInputVersion, done, err := buildCELVariablesForIssue(ctx, stores, issue)
		approvalInputVersion = celApprovalInputVersion
		if err != nil {
			return nil, nil, approvalInputVersion, false, errors.Wrap(err, "failed to build CEL variables for issue")
		}
		if !done {
			return nil, nil, approvalInputVersion, false, nil
		}

		if approvalSource == storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE {
			riskLevel := calculateRiskLevelFromCELVars(celVarsList)
			injectRiskLevelIntoCELVars(celVarsList, riskLevel)
			injectIssueLabelsIntoCELVars(celVarsList, approvalLabels)
		}

		approvalTemplate, err := getApprovalTemplate(approvalSetting, approvalSource, celVarsList)
		if err != nil {
			return nil, nil, approvalInputVersion, false, errors.Wrapf(err, "failed to get approval template for source: %v", approvalSource)
		}
		return approvalTemplate, celVarsList, approvalInputVersion, true, nil
	}()
	if err != nil {
		return err
	}
	if !done {
		return nil
	}

	issue.Payload.Approval = &storepb.IssuePayloadApproval{
		ApprovalFindingDone:  true,
		ApprovalTemplate:     approvalTemplate,
		ApprovalInputVersion: approvalInputVersion,
	}
	issue.Payload.RiskLevel = calculateRiskLevelFromCELVars(celVarsList)
	return nil
}
