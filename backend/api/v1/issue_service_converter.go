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
		v1Issue, err := s.convertToIssue(issue)
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
			if v := issueFilter.Approver; v != nil && !s.isIssueNextApprover(ctx, v1Issue, projectID, v) {
				continue
			}
		}
		converted = append(converted, v1Issue)
	}
	return converted, nil
}

func (s *IssueService) getUserRoleMap(ctx context.Context, projectResourceID string, user *store.UserMessage) map[string]bool {
	if user == nil {
		return map[string]bool{}
	}

	policy, err := s.store.GetProjectIamPolicy(ctx, projectResourceID)
	if err != nil {
		slog.Error("failed to get project iam policy", log.BBError(err), slog.String("project", projectResourceID))
		return map[string]bool{}
	}
	workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		slog.Error("failed to get workspace iam policy", log.BBError(err))
		return map[string]bool{}
	}

	return utils.GetUserFormattedRolesMap(ctx, s.store, user, policy.Policy, workspacePolicy.Policy)
}

func (s *IssueService) isIssueNextApprover(ctx context.Context, issue *v1pb.Issue, projectResourceID string, user *store.UserMessage) bool {
	if user == nil {
		return false
	}

	roles := s.getUserRoleMap(ctx, projectResourceID, user)
	approvalRoles := issue.GetApprovalTemplate().GetFlow().GetRoles()
	index := len(issue.Approvers)
	if index >= len(approvalRoles) {
		return false
	}

	return roles[approvalRoles[index]]
}

// nolint:unparam
func (*IssueService) convertToIssue(issue *store.IssueMessage) (*v1pb.Issue, error) {
	issuePayload := issue.Payload

	convertedGrantRequest := convertToGrantRequest(issuePayload.GrantRequest)

	issueV1 := &v1pb.Issue{
		Name:         common.FormatIssue(issue.ProjectID, issue.UID),
		Title:        issue.Title,
		Description:  issue.Description,
		Type:         convertToIssueType(issue.Type),
		Status:       convertToIssueStatus(issue.Status),
		Creator:      common.FormatUserEmail(issue.CreatorEmail),
		CreateTime:   timestamppb.New(issue.CreatedAt),
		UpdateTime:   timestamppb.New(issue.UpdatedAt),
		GrantRequest: convertedGrantRequest,
		Labels:       issuePayload.Labels,
	}

	if issue.PlanUID != nil {
		issueV1.Plan = common.FormatPlan(issue.ProjectID, *issue.PlanUID)
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

	return issueV1, nil
}

func computeApprovalStatus(approval *storepb.IssuePayloadApproval) v1pb.Issue_ApprovalStatus {
	// If approval finding is not done, status is checking
	// Note: approval.GetApprovalFindingDone() returns false when approval is nil
	if !approval.GetApprovalFindingDone() {
		return v1pb.Issue_CHECKING
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
	case *storepb.IssueCommentPayload_PlanSpecUpdate_:
		projectID, _, _ := common.GetProjectIDIssueUID(issueName)
		r.Event = convertToIssueCommentEventPlanSpecUpdate(projectID, e)
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

func convertToIssueCommentEventPlanSpecUpdate(projectID string, u *storepb.IssueCommentPayload_PlanSpecUpdate_) *v1pb.IssueComment_PlanSpecUpdate_ {
	result := &v1pb.IssueComment_PlanSpecUpdate_{
		PlanSpecUpdate: &v1pb.IssueComment_PlanSpecUpdate{
			Spec: u.PlanSpecUpdate.Spec,
		},
	}
	if fromSha256 := u.PlanSpecUpdate.GetFromSheetSha256(); fromSha256 != "" {
		fromSheet := common.FormatSheet(projectID, fromSha256)
		result.PlanSpecUpdate.FromSheet = &fromSheet
	}
	if toSha256 := u.PlanSpecUpdate.GetToSheetSha256(); toSha256 != "" {
		toSheet := common.FormatSheet(projectID, toSha256)
		result.PlanSpecUpdate.ToSheet = &toSheet
	}
	return result
}
