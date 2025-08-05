package v1

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *IssueService) convertToIssues(ctx context.Context, issues []*store.IssueMessage) ([]*v1pb.Issue, error) {
	var converted []*v1pb.Issue
	for _, issue := range issues {
		v1Issue, err := s.convertToIssue(ctx, issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to issue")
		}
		converted = append(converted, v1Issue)
	}
	return converted, nil
}

func (s *IssueService) convertToIssue(ctx context.Context, issue *store.IssueMessage) (*v1pb.Issue, error) {
	issuePayload := issue.Payload

	convertedGrantRequest, err := convertToGrantRequest(ctx, s.store, issuePayload.GrantRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert GrantRequest")
	}

	releasers, err := s.convertToIssueReleasers(ctx, issue)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue releasers")
	}

	issueV1 := &v1pb.Issue{
		Name:                 common.FormatIssue(issue.Project.ResourceID, issue.UID),
		Title:                issue.Title,
		Description:          issue.Description,
		Type:                 convertToIssueType(issue.Type),
		Status:               convertToIssueStatus(issue.Status),
		Approvers:            nil,
		ApprovalTemplates:    nil,
		ApprovalFindingDone:  false,
		ApprovalFindingError: "",
		Creator:              common.FormatUserEmail(issue.Creator.Email),
		CreateTime:           timestamppb.New(issue.CreatedAt),
		UpdateTime:           timestamppb.New(issue.UpdatedAt),
		Plan:                 "",
		Rollout:              "",
		GrantRequest:         convertedGrantRequest,
		Releasers:            releasers,
		RiskLevel:            v1pb.Issue_RISK_LEVEL_UNSPECIFIED,
		TaskStatusCount:      issue.TaskStatusCount,
		Labels:               issuePayload.Labels,
	}

	if issue.PlanUID != nil {
		issueV1.Plan = common.FormatPlan(issue.Project.ResourceID, *issue.PlanUID)
	}
	if issue.PipelineUID != nil {
		issueV1.Rollout = common.FormatRollout(issue.Project.ResourceID, *issue.PipelineUID)
	}

	if issuePayload.Approval != nil {
		issueV1.ApprovalFindingDone = issuePayload.Approval.ApprovalFindingDone
		issueV1.ApprovalFindingError = issuePayload.Approval.ApprovalFindingError
		issueV1.RiskLevel = convertToIssueRiskLevel(issuePayload.Approval.RiskLevel)
		for _, template := range issuePayload.Approval.ApprovalTemplates {
			issueV1.ApprovalTemplates = append(issueV1.ApprovalTemplates, convertToApprovalTemplate(template))
		}
		for _, approver := range issuePayload.Approval.Approvers {
			convertedApprover := &v1pb.Issue_Approver{Status: v1pb.Issue_Approver_Status(approver.Status)}
			user, err := s.store.GetUserByID(ctx, int(approver.PrincipalId))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find user by id %v", approver.PrincipalId)
			}
			convertedApprover.Principal = fmt.Sprintf("users/%s", user.Email)
			issueV1.Approvers = append(issueV1.Approvers, convertedApprover)
		}
	}

	return issueV1, nil
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
	for _, role := range policy.IssueRoles {
		switch role {
		case "roles/CREATOR":
			releasers = append(releasers, common.FormatUserEmail(issue.Creator.Email))
		case "roles/LAST_APPROVER":
			approvers := issue.Payload.GetApproval().GetApprovers()
			if len(approvers) > 0 {
				lastApproverUID := approvers[len(approvers)-1].GetPrincipalId()
				user, err := s.store.GetUserByID(ctx, int(lastApproverUID))
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get last approver uid %d", lastApproverUID)
				}
				releasers = append(releasers, common.FormatUserEmail(user.Email))
			}
		default:
			// Handle other roles
		}
	}

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

func convertToIssueRiskLevel(riskLevel storepb.IssuePayloadApproval_RiskLevel) v1pb.Issue_RiskLevel {
	switch riskLevel {
	case storepb.IssuePayloadApproval_RISK_LEVEL_UNSPECIFIED:
		return v1pb.Issue_RISK_LEVEL_UNSPECIFIED
	case storepb.IssuePayloadApproval_LOW:
		return v1pb.Issue_LOW
	case storepb.IssuePayloadApproval_MODERATE:
		return v1pb.Issue_MODERATE
	case storepb.IssuePayloadApproval_HIGH:
		return v1pb.Issue_HIGH
	default:
		return v1pb.Issue_RISK_LEVEL_UNSPECIFIED
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
	convertedFlow := &v1pb.ApprovalFlow{}
	for _, step := range flow.Steps {
		convertedFlow.Steps = append(convertedFlow.Steps, convertToApprovalStep(step))
	}
	return convertedFlow
}

func convertToApprovalStep(step *storepb.ApprovalStep) *v1pb.ApprovalStep {
	convertedStep := &v1pb.ApprovalStep{
		Type: v1pb.ApprovalStep_Type(step.Type),
	}
	for _, node := range step.Nodes {
		convertedStep.Nodes = append(convertedStep.Nodes, convertToApprovalNode(node))
	}
	return convertedStep
}

func convertToApprovalNode(node *storepb.ApprovalNode) *v1pb.ApprovalNode {
	v1node := &v1pb.ApprovalNode{}
	v1node.Type = v1pb.ApprovalNode_Type(node.Type)
	v1node.Role = node.Role
	return v1node
}

func convertToGrantRequest(ctx context.Context, s *store.Store, v *storepb.GrantRequest) (*v1pb.GrantRequest, error) {
	if v == nil {
		return nil, nil
	}
	uid, err := common.GetUserID(v.User)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user uid from %q", v.User)
	}
	user, err := s.GetUserByID(ctx, uid)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user by uid %q", uid)
	}
	if user == nil {
		return nil, errors.Errorf("user %q not found", v.User)
	}
	return &v1pb.GrantRequest{
		Role:       v.Role,
		User:       common.FormatUserEmail(user.Email),
		Condition:  v.Condition,
		Expiration: v.Expiration,
	}, nil
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
		User:       common.FormatUserUID(user.ID),
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
		Creator:    common.FormatUserEmail(ic.Creator.Email),
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

func convertToIssueCommentPayloadIssueUpdateIssueStatus(s *v1pb.IssueStatus) *storepb.IssueCommentPayload_IssueUpdate_IssueStatus {
	if s == nil {
		return nil
	}
	var is storepb.IssueCommentPayload_IssueUpdate_IssueStatus
	switch *s {
	case v1pb.IssueStatus_CANCELED:
		is = storepb.IssueCommentPayload_IssueUpdate_CANCELED
	case v1pb.IssueStatus_DONE:
		is = storepb.IssueCommentPayload_IssueUpdate_DONE
	case v1pb.IssueStatus_OPEN:
		is = storepb.IssueCommentPayload_IssueUpdate_OPEN
	case v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED:
		is = storepb.IssueCommentPayload_IssueUpdate_ISSUE_STATUS_UNSPECIFIED
	default:
		is = storepb.IssueCommentPayload_IssueUpdate_ISSUE_STATUS_UNSPECIFIED
	}
	return &is
}

func convertToIssueCommentEventIssueUpdateStatus(s *storepb.IssueCommentPayload_IssueUpdate_IssueStatus) *v1pb.IssueStatus {
	if s == nil {
		return nil
	}
	var is v1pb.IssueStatus
	switch *s {
	case storepb.IssueCommentPayload_IssueUpdate_CANCELED:
		is = v1pb.IssueStatus_CANCELED
	case storepb.IssueCommentPayload_IssueUpdate_DONE:
		is = v1pb.IssueStatus_DONE
	case storepb.IssueCommentPayload_IssueUpdate_OPEN:
		is = v1pb.IssueStatus_OPEN
	case storepb.IssueCommentPayload_IssueUpdate_ISSUE_STATUS_UNSPECIFIED:
		is = v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED
	default:
		is = v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED
	}
	return &is
}

func convertToIssueCommentEventApprovalStatus(s storepb.IssueCommentPayload_Approval_Status) v1pb.IssueComment_Approval_Status {
	switch s {
	case storepb.IssueCommentPayload_Approval_APPROVED:
		return v1pb.IssueComment_Approval_APPROVED
	case storepb.IssueCommentPayload_Approval_PENDING:
		return v1pb.IssueComment_Approval_PENDING
	case storepb.IssueCommentPayload_Approval_REJECTED:
		return v1pb.IssueComment_Approval_REJECTED
	case storepb.IssueCommentPayload_Approval_STATUS_UNSPECIFIED:
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

func convertToIssueCommentEventTaskUpdateStatus(s *storepb.IssueCommentPayload_TaskUpdate_Status) *v1pb.IssueComment_TaskUpdate_Status {
	if s == nil {
		return nil
	}
	var r v1pb.IssueComment_TaskUpdate_Status
	//exhaustive:enforce
	switch *s {
	case storepb.IssueCommentPayload_TaskUpdate_DONE:
		r = v1pb.IssueComment_TaskUpdate_DONE
	case storepb.IssueCommentPayload_TaskUpdate_CANCELED:
		r = v1pb.IssueComment_TaskUpdate_CANCELED
	case storepb.IssueCommentPayload_TaskUpdate_FAILED:
		r = v1pb.IssueComment_TaskUpdate_FAILED
	case storepb.IssueCommentPayload_TaskUpdate_PENDING:
		r = v1pb.IssueComment_TaskUpdate_PENDING
	case storepb.IssueCommentPayload_TaskUpdate_RUNNING:
		r = v1pb.IssueComment_TaskUpdate_RUNNING
	case storepb.IssueCommentPayload_TaskUpdate_SKIPPED:
		r = v1pb.IssueComment_TaskUpdate_SKIPPED
	case storepb.IssueCommentPayload_TaskUpdate_STATUS_UNSPECIFIED:
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
