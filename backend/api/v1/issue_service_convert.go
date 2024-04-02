package v1

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func convertToIssues(ctx context.Context, s *store.Store, issues []*store.IssueMessage) ([]*v1pb.Issue, error) {
	var converted []*v1pb.Issue
	for _, issue := range issues {
		v1Issue, err := convertToIssue(ctx, s, issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to issue")
		}
		converted = append(converted, v1Issue)
	}
	return converted, nil
}

func convertToIssue(ctx context.Context, s *store.Store, issue *store.IssueMessage) (*v1pb.Issue, error) {
	issuePayload := issue.Payload

	convertedGrantRequest, err := convertToGrantRequest(ctx, s, issuePayload.GrantRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert GrantRequest")
	}

	releasers, err := convertToIssueReleasers(ctx, s, issue)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue releasers")
	}

	issueV1 := &v1pb.Issue{
		Name:                 fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, issue.Project.ResourceID, common.IssueNamePrefix, issue.UID),
		Uid:                  fmt.Sprintf("%d", issue.UID),
		Title:                issue.Title,
		Description:          issue.Description,
		Type:                 convertToIssueType(issue.Type),
		Status:               convertToIssueStatus(issue.Status),
		Assignee:             "",
		Approvers:            nil,
		ApprovalTemplates:    nil,
		ApprovalFindingDone:  false,
		ApprovalFindingError: "",
		Subscribers:          nil,
		Creator:              fmt.Sprintf("%s%s", common.UserNamePrefix, issue.Creator.Email),
		CreateTime:           timestamppb.New(issue.CreatedTime),
		UpdateTime:           timestamppb.New(issue.UpdatedTime),
		Plan:                 "",
		Rollout:              "",
		GrantRequest:         convertedGrantRequest,
		Releasers:            releasers,
		RiskLevel:            v1pb.Issue_RISK_LEVEL_UNSPECIFIED,
		TaskStatusCount:      issue.TaskStatusCount,
	}

	if issue.PlanUID != nil {
		issueV1.Plan = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, issue.Project.ResourceID, common.PlanPrefix, *issue.PlanUID)
	}
	if issue.PipelineUID != nil {
		issueV1.Rollout = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, issue.Project.ResourceID, common.RolloutPrefix, *issue.PipelineUID)
	}
	if issue.Assignee != nil {
		issueV1.Assignee = fmt.Sprintf("%s%s", common.UserNamePrefix, issue.Assignee.Email)
	}

	for _, subscriber := range issue.Subscribers {
		issueV1.Subscribers = append(issueV1.Subscribers, fmt.Sprintf("%s%s", common.UserNamePrefix, subscriber.Email))
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
			user, err := s.GetUserByID(ctx, int(approver.PrincipalId))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find user by id %v", approver.PrincipalId)
			}
			convertedApprover.Principal = fmt.Sprintf("users/%s", user.Email)
			issueV1.Approvers = append(issueV1.Approvers, convertedApprover)
		}
	}

	return issueV1, nil
}

func convertToIssueReleasers(ctx context.Context, s *store.Store, issue *store.IssueMessage) ([]string, error) {
	if issue.Type != api.IssueDatabaseGeneral {
		return nil, nil
	}
	if issue.Status != api.IssueOpen {
		return nil, nil
	}
	if issue.PipelineUID == nil {
		return nil, nil
	}
	stages, err := s.ListStageV2(ctx, *issue.PipelineUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list issue stages")
	}
	var activeStage *store.StageMessage
	for _, stage := range stages {
		if stage.Active {
			activeStage = stage
			break
		}
	}
	if activeStage == nil {
		return nil, nil
	}
	policy, err := s.GetRolloutPolicy(ctx, activeStage.EnvironmentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rollout policy")
	}

	var releasers []string
	if policy.Automatic {
		releasers = append(releasers, common.FormatRole(api.ProjectOwner.String()), common.FormatUserEmail(issue.Creator.Email))
		return releasers, nil
	}

	releasers = append(releasers, policy.WorkspaceRoles...)
	releasers = append(releasers, policy.ProjectRoles...)

	for _, role := range policy.IssueRoles {
		switch role {
		case "roles/CREATOR":
			releasers = append(releasers, common.FormatUserEmail(issue.Creator.Email))
		case "roles/LAST_APPROVER":
			approvers := issue.Payload.GetApproval().GetApprovers()
			if len(approvers) > 0 {
				lastApproverUID := approvers[len(approvers)-1].GetPrincipalId()
				user, err := s.GetUserByID(ctx, int(lastApproverUID))
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get last approver uid %d", lastApproverUID)
				}
				releasers = append(releasers, common.FormatUserEmail(user.Email))
			}
		}
	}

	return releasers, nil
}

func convertToIssueType(t api.IssueType) v1pb.Issue_Type {
	switch t {
	case api.IssueDatabaseGeneral:
		return v1pb.Issue_DATABASE_CHANGE
	case api.IssueGrantRequest:
		return v1pb.Issue_GRANT_REQUEST
	case api.IssueDatabaseDataExport:
		return v1pb.Issue_DATABASE_DATA_EXPORT
	default:
		return v1pb.Issue_TYPE_UNSPECIFIED
	}
}

func convertToAPIIssueStatus(status v1pb.IssueStatus) (api.IssueStatus, error) {
	switch status {
	case v1pb.IssueStatus_OPEN:
		return api.IssueOpen, nil
	case v1pb.IssueStatus_DONE:
		return api.IssueDone, nil
	case v1pb.IssueStatus_CANCELED:
		return api.IssueCanceled, nil
	default:
		return api.IssueStatus(""), errors.Errorf("invalid issue status %v", status)
	}
}

func convertToIssueStatus(status api.IssueStatus) v1pb.IssueStatus {
	switch status {
	case api.IssueOpen:
		return v1pb.IssueStatus_OPEN
	case api.IssueDone:
		return v1pb.IssueStatus_DONE
	case api.IssueCanceled:
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
	switch payload := node.Payload.(type) {
	case *storepb.ApprovalNode_GroupValue_:
		v1node.Payload = &v1pb.ApprovalNode_GroupValue_{
			GroupValue: convertToApprovalNodeGroupValue(payload.GroupValue),
		}
	case *storepb.ApprovalNode_Role:
		v1node.Payload = &v1pb.ApprovalNode_Role{
			Role: payload.Role,
		}
	case *storepb.ApprovalNode_ExternalNodeId:
		v1node.Payload = &v1pb.ApprovalNode_ExternalNodeId{
			ExternalNodeId: payload.ExternalNodeId,
		}
	}
	return v1node
}

func convertToApprovalNodeGroupValue(v storepb.ApprovalNode_GroupValue) v1pb.ApprovalNode_GroupValue {
	switch v {
	case storepb.ApprovalNode_GROUP_VALUE_UNSPECIFILED:
		return v1pb.ApprovalNode_GROUP_VALUE_UNSPECIFILED
	case storepb.ApprovalNode_WORKSPACE_OWNER:
		return v1pb.ApprovalNode_WORKSPACE_OWNER
	case storepb.ApprovalNode_WORKSPACE_DBA:
		return v1pb.ApprovalNode_WORKSPACE_DBA
	case storepb.ApprovalNode_PROJECT_OWNER:
		return v1pb.ApprovalNode_PROJECT_OWNER
	case storepb.ApprovalNode_PROJECT_MEMBER:
		return v1pb.ApprovalNode_PROJECT_MEMBER
	default:
		return v1pb.ApprovalNode_GROUP_VALUE_UNSPECIFILED
	}
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
	user, err := s.GetUser(ctx, &store.FindUserMessage{Email: &email})
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
		Uid:        strconv.Itoa(ic.UID),
		Comment:    ic.Payload.Comment,
		CreateTime: timestamppb.New(time.Unix(ic.CreatedTs, 0)),
		UpdateTime: timestamppb.New(time.Unix(ic.UpdatedTs, 0)),
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
	case *storepb.IssueCommentPayload_TaskRunUpdate_:
		r.Event = convertToIssueCommentEventTaskRunUpdate(e)
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
			FromAssignee:    u.IssueUpdate.FromAssignee,
			ToAssignee:      u.IssueUpdate.ToAssignee,
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

func convertToIssueCommentEventTaskRunUpdate(u *storepb.IssueCommentPayload_TaskRunUpdate_) *v1pb.IssueComment_TaskRunUpdate_ {
	return &v1pb.IssueComment_TaskRunUpdate_{
		TaskRunUpdate: &v1pb.IssueComment_TaskRunUpdate{
			TaskRuns: u.TaskRunUpdate.TaskRuns,
			ToStatus: convertToIssueCommentEventTaskRunUpdateStatus(u.TaskRunUpdate.ToStatus),
		},
	}
}

func convertToIssueCommentEventTaskRunUpdateStatus(s *storepb.IssueCommentPayload_TaskRunUpdate_Status) *v1pb.IssueComment_TaskRunUpdate_Status {
	if s == nil {
		return nil
	}
	var r v1pb.IssueComment_TaskRunUpdate_Status
	switch *s {
	case storepb.IssueCommentPayload_TaskRunUpdate_DONE:
		r = v1pb.IssueComment_TaskRunUpdate_DONE
	case storepb.IssueCommentPayload_TaskRunUpdate_FAILED:
		r = v1pb.IssueComment_TaskRunUpdate_FAILED
	case storepb.IssueCommentPayload_TaskRunUpdate_PENDING:
		r = v1pb.IssueComment_TaskRunUpdate_PENDING
	case storepb.IssueCommentPayload_TaskRunUpdate_RUNNING:
		r = v1pb.IssueComment_TaskRunUpdate_RUNNING
	case storepb.IssueCommentPayload_TaskRunUpdate_STATUS_UNSPECIFIED:
		r = v1pb.IssueComment_TaskRunUpdate_STATUS_UNSPECIFIED
	default:
		r = v1pb.IssueComment_TaskRunUpdate_STATUS_UNSPECIFIED
	}
	return &r
}

func convertToIssueCommentEventTaskUpdate(u *storepb.IssueCommentPayload_TaskUpdate_) *v1pb.IssueComment_TaskUpdate_ {
	return &v1pb.IssueComment_TaskUpdate_{
		TaskUpdate: &v1pb.IssueComment_TaskUpdate{
			Tasks:                   u.TaskUpdate.Tasks,
			FromSheet:               u.TaskUpdate.FromSheet,
			ToSheet:                 u.TaskUpdate.ToSheet,
			FromEarliestAllowedTime: u.TaskUpdate.FromEarliestAllowedTime,
			ToEarliestAllowedTime:   u.TaskUpdate.ToEarliestAllowedTime,
		},
	}
}

func convertToIssueCommentEventTaskPriorBackup(b *storepb.IssueCommentPayload_TaskPriorBackup_) *v1pb.IssueComment_TaskPriorBackup_ {
	return &v1pb.IssueComment_TaskPriorBackup_{
		TaskPriorBackup: &v1pb.IssueComment_TaskPriorBackup{
			Task:   b.TaskPriorBackup.Task,
			Tables: convertToIssueCommentEventTaskPriorBackupTables(b.TaskPriorBackup.Tables),
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
