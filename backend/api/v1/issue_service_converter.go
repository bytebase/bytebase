package v1

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

func (s *IssueService) convertToIssues(ctx context.Context, issues []*store.IssueMessage, issueFilter *filterIssueMessage) ([]*v1pb.Issue, error) {
	var converted []*v1pb.Issue
	for _, issue := range issues {
		v1Issue, err := s.convertToIssue(ctx, issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to issue")
		}
		if v1Issue == nil {
			continue
		}
		if issueFilter != nil {
			if v := issueFilter.ApprovalStatus; v != nil && v1Issue.ApprovalStatus != *v {
				continue
			}
			projectID, _, err := common.GetProjectIDIssueUID(v1Issue.Name)
			if err != nil {
				slog.Error("failed to parse the issue name", log.BBError(err), slog.String("issue", v1Issue.Name))
				continue
			}
			if v := issueFilter.ApproverID; v != nil && !s.isIssueNextApprover(ctx, v1Issue, projectID, *v) {
				continue
			}
		}
		converted = append(converted, v1Issue)
	}
	return converted, nil
}

func (s *IssueService) getUserRoleMap(ctx context.Context, projectResourceID string, principalUID int) map[string]bool {
	policy, err := s.store.GetProjectIamPolicy(ctx, projectResourceID)
	slog.Error("failed to get project iam policy", log.BBError(err), slog.String("project", projectResourceID))
	if err != nil {
		return map[string]bool{}
	}
	workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		slog.Error("failed to get workspace iam policy", log.BBError(err))
		return map[string]bool{}
	}

	user, err := s.store.GetUserByID(ctx, principalUID)
	if err != nil {
		slog.Error("failed to get user", log.BBError(err), slog.Int("user_id", principalUID))
		return map[string]bool{}
	}
	if user == nil {
		return map[string]bool{}
	}

	return utils.GetUserFormattedRolesMap(ctx, s.store, user, policy.Policy, workspacePolicy.Policy)
}

func (s *IssueService) isIssueNextApprover(ctx context.Context, issue *v1pb.Issue, projectResourceID string, principalUID int) bool {
	roles := s.getUserRoleMap(ctx, projectResourceID, principalUID)
	approvalRoles := issue.GetApprovalTemplate().GetFlow().GetRoles()
	index := len(issue.Approvers)
	if index >= len(approvalRoles) {
		return false
	}

	return roles[approvalRoles[index]]
}

func (s *IssueService) convertToIssue(ctx context.Context, issue *store.IssueMessage) (*v1pb.Issue, error) {
	issuePayload := issue.Payload

	convertedGrantRequest := convertToGrantRequest(issuePayload.GrantRequest)

	releasers, err := s.convertToIssueReleasers(ctx, issue)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue releasers")
	}

	issueV1 := &v1pb.Issue{
		Name:            common.FormatIssue(issue.Project.ResourceID, issue.UID),
		Title:           issue.Title,
		Description:     issue.Description,
		Type:            convertToIssueType(issue.Type),
		Status:          convertToIssueStatus(issue.Status),
		Creator:         common.FormatUserEmail(issue.CreatorEmail),
		CreateTime:      timestamppb.New(issue.CreatedAt),
		UpdateTime:      timestamppb.New(issue.UpdatedAt),
		GrantRequest:    convertedGrantRequest,
		Releasers:       releasers,
		TaskStatusCount: issue.TaskStatusCount,
		Labels:          issuePayload.Labels,
	}

	if issue.PlanUID != nil {
		issueV1.Plan = common.FormatPlan(issue.Project.ResourceID, *issue.PlanUID)
	}
	if issue.PipelineUID != nil {
		issueV1.Rollout = common.FormatRollout(issue.Project.ResourceID, *issue.PipelineUID)
	}

	approval := issuePayload.GetApproval()
	issueV1.RiskLevel = convertToIssueRiskLevel(issuePayload.GetRiskLevel())
	if template := approval.GetApprovalTemplate(); template != nil {
		issueV1.ApprovalTemplate = convertToApprovalTemplate(template)
	}
	for _, approver := range approval.GetApprovers() {
		convertedApprover := &v1pb.Issue_Approver{
			Status:    v1pb.Issue_Approver_Status(approver.GetStatus()),
			Principal: approver.GetPrincipal(),
		}
		issueV1.Approvers = append(issueV1.Approvers, convertedApprover)
	}
	issueV1.ApprovalStatus = computeApprovalStatus(approval)
	issueV1.ApprovalStatusError = approval.GetApprovalFindingError()

	return issueV1, nil
}

func computeApprovalStatus(approval *storepb.IssuePayloadApproval) v1pb.Issue_ApprovalStatus {
	// If approval finding is not done, status is checking
	// Note: approval.GetApprovalFindingDone() returns false when approval is nil
	if !approval.GetApprovalFindingDone() {
		return v1pb.Issue_CHECKING
	}

	// If there's an error finding approval, status is error
	if approval.GetApprovalFindingError() != "" {
		return v1pb.Issue_ERROR
	}

	// If no approval template, approval is skipped (not required)
	if approval.GetApprovalTemplate() == nil {
		return v1pb.Issue_SKIPPED
	}

	approvalTemplate := approval.GetApprovalTemplate()
	approvers := approval.GetApprovers()
	totalSteps := len(approvalTemplate.GetFlow().GetRoles())

	// If no approvers are assigned yet, it's pending
	if len(approvers) == 0 {
		return v1pb.Issue_PENDING
	}

	// Check approver statuses
	for _, approver := range approvers {
		if approver.GetStatus() == storepb.IssuePayloadApproval_Approver_REJECTED {
			// Short-circuit: if any approver rejected, overall status is rejected
			return v1pb.Issue_REJECTED
		}
	}

	// Check if all steps are completed
	// Each approver corresponds to one step in the approval flow
	// All steps are approved if:
	// 1. Number of approvers equals number of steps
	// 2. All approvers have APPROVED status
	if len(approvers) >= totalSteps {
		allApproved := true
		for _, approver := range approvers {
			if approver.GetStatus() != storepb.IssuePayloadApproval_Approver_APPROVED {
				allApproved = false
				break
			}
		}
		if allApproved {
			return v1pb.Issue_APPROVED
		}
	}

	// Otherwise, approval is pending (more steps to complete or waiting for approvals)
	return v1pb.Issue_PENDING
}

func (s *IssueService) convertToIssueReleasers(ctx context.Context, issue *store.IssueMessage) ([]string, error) {
	if issue.Type != storepb.Issue_DATABASE_CHANGE {
		return nil, nil
	}
	if issue.Status != storepb.Issue_OPEN {
		return nil, nil
	}
	if issue.PipelineUID == nil {
		return nil, nil
	}
	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: issue.PipelineUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list issue tasks")
	}
	// Find the active environment (first environment with non-completed tasks)
	var activeEnvironment string
	for _, task := range tasks {
		if task.LatestTaskRunStatus != storepb.TaskRun_DONE && task.LatestTaskRunStatus != storepb.TaskRun_SKIPPED {
			activeEnvironment = task.Environment
			break
		}
	}
	if activeEnvironment == "" {
		return nil, nil
	}
	policy, err := GetValidRolloutPolicyForEnvironment(ctx, s.store, activeEnvironment)
	if err != nil {
		return nil, err
	}

	var releasers []string

	releasers = append(releasers, policy.Roles...)

	return releasers, nil
}

func convertToIssueType(t storepb.Issue_Type) v1pb.Issue_Type {
	switch t {
	case storepb.Issue_DATABASE_CHANGE:
		return v1pb.Issue_DATABASE_CHANGE
	case storepb.Issue_GRANT_REQUEST:
		return v1pb.Issue_GRANT_REQUEST
	case storepb.Issue_DATABASE_EXPORT:
		return v1pb.Issue_DATABASE_EXPORT
	default:
		return v1pb.Issue_TYPE_UNSPECIFIED
	}
}

func convertToAPIIssueType(t v1pb.Issue_Type) (storepb.Issue_Type, error) {
	switch t {
	case v1pb.Issue_DATABASE_CHANGE:
		return storepb.Issue_DATABASE_CHANGE, nil
	case v1pb.Issue_GRANT_REQUEST:
		return storepb.Issue_GRANT_REQUEST, nil
	case v1pb.Issue_DATABASE_EXPORT:
		return storepb.Issue_DATABASE_EXPORT, nil
	default:
		return storepb.Issue_ISSUE_TYPE_UNSPECIFIED, errors.Errorf("invalid issue type %v", t)
	}
}

func convertToAPIIssueStatus(status v1pb.IssueStatus) (storepb.Issue_Status, error) {
	switch status {
	case v1pb.IssueStatus_OPEN:
		return storepb.Issue_OPEN, nil
	case v1pb.IssueStatus_DONE:
		return storepb.Issue_DONE, nil
	case v1pb.IssueStatus_CANCELED:
		return storepb.Issue_CANCELED, nil
	default:
		return storepb.Issue_ISSUE_STATUS_UNSPECIFIED, errors.Errorf("invalid issue status %v", status)
	}
}

func convertToIssueStatus(status storepb.Issue_Status) v1pb.IssueStatus {
	switch status {
	case storepb.Issue_OPEN:
		return v1pb.IssueStatus_OPEN
	case storepb.Issue_DONE:
		return v1pb.IssueStatus_DONE
	case storepb.Issue_CANCELED:
		return v1pb.IssueStatus_CANCELED
	default:
		return v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED
	}
}

func convertToIssueRiskLevel(riskLevel storepb.RiskLevel) v1pb.RiskLevel {
	switch riskLevel {
	case storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED:
		return v1pb.RiskLevel_RISK_LEVEL_UNSPECIFIED
	case storepb.RiskLevel_LOW:
		return v1pb.RiskLevel_LOW
	case storepb.RiskLevel_MODERATE:
		return v1pb.RiskLevel_MODERATE
	case storepb.RiskLevel_HIGH:
		return v1pb.RiskLevel_HIGH
	default:
		return v1pb.RiskLevel_RISK_LEVEL_UNSPECIFIED
	}
}

func convertToApprovalTemplate(template *storepb.ApprovalTemplate) *v1pb.ApprovalTemplate {
	return &v1pb.ApprovalTemplate{
		Flow:        convertToApprovalFlow(template.Flow),
		Title:       template.Title,
		Description: template.Description,
	}
}

func convertToApprovalFlow(flow *storepb.ApprovalFlow) *v1pb.ApprovalFlow {
	return &v1pb.ApprovalFlow{
		Roles: flow.Roles,
	}
}

func convertToGrantRequest(v *storepb.GrantRequest) *v1pb.GrantRequest {
	if v == nil {
		return nil
	}
	return &v1pb.GrantRequest{
		Role:       v.Role,
		User:       v.User,
		Condition:  v.Condition,
		Expiration: v.Expiration,
	}
}

func convertGrantRequest(ctx context.Context, s *store.Store, v *v1pb.GrantRequest) (*storepb.GrantRequest, error) {
	if v == nil {
		return nil, nil
	}
	email, err := common.GetUserEmail(v.User)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user email from %q", v.User)
	}
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user by email %q", email)
	}
	if user == nil {
		return nil, errors.Errorf("user %q not found", v.User)
	}
	return &storepb.GrantRequest{
		Role:       v.Role,
		User:       common.FormatUserEmail(user.Email),
		Condition:  v.Condition,
		Expiration: v.Expiration,
	}, nil
}

func convertToIssueComments(issueName string, issueComments []*store.IssueCommentMessage) []*v1pb.IssueComment {
	var res []*v1pb.IssueComment
	for _, ic := range issueComments {
		res = append(res, convertToIssueComment(issueName, ic))
	}
	return res
}

func convertToIssueComment(issueName string, ic *store.IssueCommentMessage) *v1pb.IssueComment {
	r := &v1pb.IssueComment{
		Comment:    ic.Payload.Comment,
		CreateTime: timestamppb.New(ic.CreatedAt),
		UpdateTime: timestamppb.New(ic.UpdatedAt),
		Name:       fmt.Sprintf("%s/%s%d", issueName, common.IssueCommentNamePrefix, ic.UID),
		Creator:    common.FormatUserEmail(ic.CreatorEmail),
	}

	switch e := ic.Payload.Event.(type) {
	case *storepb.IssueCommentPayload_Approval_:
		r.Event = convertToIssueCommentEventApproval(e)
	case *storepb.IssueCommentPayload_IssueUpdate_:
		r.Event = convertToIssueCommentEventIssueUpdate(e)
	case *storepb.IssueCommentPayload_StageEnd_:
		r.Event = convertToIssueCommentEventStageEnd(e)
	case *storepb.IssueCommentPayload_TaskUpdate_:
		r.Event = convertToIssueCommentEventTaskUpdate(e)
	case *storepb.IssueCommentPayload_TaskPriorBackup_:
		r.Event = convertToIssueCommentEventTaskPriorBackup(e)
	}

	return r
}

func convertToIssueCommentEventApproval(a *storepb.IssueCommentPayload_Approval_) *v1pb.IssueComment_Approval_ {
	return &v1pb.IssueComment_Approval_{
		Approval: &v1pb.IssueComment_Approval{
			Status: convertToIssueCommentEventApprovalStatus(a.Approval.Status),
		},
	}
}

func convertToIssueCommentEventIssueUpdate(u *storepb.IssueCommentPayload_IssueUpdate_) *v1pb.IssueComment_IssueUpdate_ {
	return &v1pb.IssueComment_IssueUpdate_{
		IssueUpdate: &v1pb.IssueComment_IssueUpdate{
			FromTitle:       u.IssueUpdate.FromTitle,
			ToTitle:         u.IssueUpdate.ToTitle,
			FromDescription: u.IssueUpdate.FromDescription,
			ToDescription:   u.IssueUpdate.ToDescription,
			FromStatus:      convertToIssueCommentEventIssueUpdateStatus(u.IssueUpdate.FromStatus),
			ToStatus:        convertToIssueCommentEventIssueUpdateStatus(u.IssueUpdate.ToStatus),
			FromLabels:      u.IssueUpdate.FromLabels,
			ToLabels:        u.IssueUpdate.ToLabels,
		},
	}
}

func convertToIssueCommentEventStageEnd(e *storepb.IssueCommentPayload_StageEnd_) *v1pb.IssueComment_StageEnd_ {
	return &v1pb.IssueComment_StageEnd_{
		StageEnd: &v1pb.IssueComment_StageEnd{
			Stage: e.StageEnd.Stage,
		},
	}
}

func convertToIssueCommentPayloadIssueUpdateIssueStatus(s *v1pb.IssueStatus) *storepb.Issue_Status {
	if s == nil {
		return nil
	}
	var is storepb.Issue_Status
	switch *s {
	case v1pb.IssueStatus_CANCELED:
		is = storepb.Issue_CANCELED
	case v1pb.IssueStatus_DONE:
		is = storepb.Issue_DONE
	case v1pb.IssueStatus_OPEN:
		is = storepb.Issue_OPEN
	case v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED:
		is = storepb.Issue_ISSUE_STATUS_UNSPECIFIED
	default:
		is = storepb.Issue_ISSUE_STATUS_UNSPECIFIED
	}
	return &is
}

func convertToIssueCommentEventIssueUpdateStatus(s *storepb.Issue_Status) *v1pb.IssueStatus {
	if s == nil {
		return nil
	}
	var is v1pb.IssueStatus
	switch *s {
	case storepb.Issue_CANCELED:
		is = v1pb.IssueStatus_CANCELED
	case storepb.Issue_DONE:
		is = v1pb.IssueStatus_DONE
	case storepb.Issue_OPEN:
		is = v1pb.IssueStatus_OPEN
	case storepb.Issue_ISSUE_STATUS_UNSPECIFIED:
		is = v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED
	default:
		is = v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED
	}
	return &is
}

func convertToIssueCommentEventApprovalStatus(s storepb.IssuePayloadApproval_Approver_Status) v1pb.IssueComment_Approval_Status {
	switch s {
	case storepb.IssuePayloadApproval_Approver_APPROVED:
		return v1pb.IssueComment_Approval_APPROVED
	case storepb.IssuePayloadApproval_Approver_PENDING:
		return v1pb.IssueComment_Approval_PENDING
	case storepb.IssuePayloadApproval_Approver_REJECTED:
		return v1pb.IssueComment_Approval_REJECTED
	case storepb.IssuePayloadApproval_Approver_STATUS_UNSPECIFIED:
		return v1pb.IssueComment_Approval_STATUS_UNSPECIFIED
	default:
		return v1pb.IssueComment_Approval_STATUS_UNSPECIFIED
	}
}

func convertToIssueCommentEventTaskUpdate(u *storepb.IssueCommentPayload_TaskUpdate_) *v1pb.IssueComment_TaskUpdate_ {
	return &v1pb.IssueComment_TaskUpdate_{
		TaskUpdate: &v1pb.IssueComment_TaskUpdate{
			Tasks:     u.TaskUpdate.Tasks,
			FromSheet: u.TaskUpdate.FromSheet,
			ToSheet:   u.TaskUpdate.ToSheet,
			ToStatus:  convertToIssueCommentEventTaskUpdateStatus(u.TaskUpdate.ToStatus),
		},
	}
}

func convertToIssueCommentEventTaskUpdateStatus(s *storepb.TaskRun_Status) *v1pb.IssueComment_TaskUpdate_Status {
	if s == nil {
		return nil
	}
	var r v1pb.IssueComment_TaskUpdate_Status
	//exhaustive:enforce
	switch *s {
	case storepb.TaskRun_DONE:
		r = v1pb.IssueComment_TaskUpdate_DONE
	case storepb.TaskRun_CANCELED:
		r = v1pb.IssueComment_TaskUpdate_CANCELED
	case storepb.TaskRun_FAILED:
		r = v1pb.IssueComment_TaskUpdate_FAILED
	case storepb.TaskRun_PENDING:
		r = v1pb.IssueComment_TaskUpdate_PENDING
	case storepb.TaskRun_RUNNING:
		r = v1pb.IssueComment_TaskUpdate_RUNNING
	case storepb.TaskRun_SKIPPED:
		r = v1pb.IssueComment_TaskUpdate_SKIPPED
	case storepb.TaskRun_NOT_STARTED:
		r = v1pb.IssueComment_TaskUpdate_STATUS_UNSPECIFIED
	case storepb.TaskRun_STATUS_UNSPECIFIED:
		r = v1pb.IssueComment_TaskUpdate_STATUS_UNSPECIFIED
	default:
		r = v1pb.IssueComment_TaskUpdate_STATUS_UNSPECIFIED
	}
	return &r
}

func convertToIssueCommentEventTaskPriorBackup(b *storepb.IssueCommentPayload_TaskPriorBackup_) *v1pb.IssueComment_TaskPriorBackup_ {
	return &v1pb.IssueComment_TaskPriorBackup_{
		TaskPriorBackup: &v1pb.IssueComment_TaskPriorBackup{
			Task:         b.TaskPriorBackup.Task,
			Tables:       convertToIssueCommentEventTaskPriorBackupTables(b.TaskPriorBackup.Tables),
			Database:     b.TaskPriorBackup.Database,
			OriginalLine: b.TaskPriorBackup.OriginalLine,
			Error:        b.TaskPriorBackup.Error,
		},
	}
}

func convertToIssueCommentEventTaskPriorBackupTables(tables []*storepb.IssueCommentPayload_TaskPriorBackup_Table) (r []*v1pb.IssueComment_TaskPriorBackup_Table) {
	for _, t := range tables {
		r = append(r, &v1pb.IssueComment_TaskPriorBackup_Table{
			Schema: t.Schema,
			Table:  t.Table,
		})
	}
	return r
}
