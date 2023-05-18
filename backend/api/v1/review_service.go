package v1

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/runner/taskcheck"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ReviewService implements the review service.
type ReviewService struct {
	v1pb.UnimplementedReviewServiceServer
	store              *store.Store
	activityManager    *activity.Manager
	taskScheduler      *taskrun.Scheduler
	taskCheckScheduler *taskcheck.Scheduler
	stateCfg           *state.State
}

// NewReviewService creates a new ReviewService.
func NewReviewService(store *store.Store, activityManager *activity.Manager, taskScheduler *taskrun.Scheduler, stateCfg *state.State) *ReviewService {
	return &ReviewService{
		store:           store,
		activityManager: activityManager,
		taskScheduler:   taskScheduler,
		stateCfg:        stateCfg,
	}
}

// GetReview gets a review.
// Currently, only review.ApprovalTemplates and review.Approvers are set.
func (s *ReviewService) GetReview(ctx context.Context, request *v1pb.GetReviewRequest) (*v1pb.Review, error) {
	reviewID, err := getReviewID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &reviewID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue, error: %v", err)
	}
	if issue == nil {
		return nil, status.Errorf(codes.NotFound, "issue %d not found", reviewID)
	}
	review, err := convertToReview(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to review, error: %v", err)
	}
	return review, nil
}

// ApproveReview approves the approval flow of the review.
func (s *ReviewService) ApproveReview(ctx context.Context, request *v1pb.ApproveReviewRequest) (*v1pb.Review, error) {
	reviewID, err := getReviewID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &reviewID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue, error: %v", err)
	}
	if issue == nil {
		return nil, status.Errorf(codes.NotFound, "issue %d not found", reviewID)
	}
	payload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal issue payload, error: %v", err)
	}
	if payload.Approval == nil {
		return nil, status.Errorf(codes.Internal, "issue payload approval is nil")
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding is not done")
	}
	if payload.Approval.ApprovalFindingError != "" {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding failed: %v", payload.Approval.ApprovalFindingError)
	}
	if len(payload.Approval.ApprovalTemplates) != 1 {
		return nil, status.Errorf(codes.Internal, "expecting one approval template but got %v", len(payload.Approval.ApprovalTemplates))
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return nil, status.Errorf(codes.InvalidArgument, "the review has been approved")
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user by id %v", principalID)
	}

	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project policy, error: %v", err)
	}

	canApprove, err := canUserApproveStep(step, user, policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if principal can approve step, error: %v", err)
	}
	if !canApprove {
		return nil, status.Errorf(codes.PermissionDenied, "cannot approve because the user does not have the required permission")
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:      storepb.IssuePayloadApproval_Approver_APPROVED,
		PrincipalId: int32(principalID),
	})

	approved, err := utils.CheckApprovalApproved(payload.Approval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the approval is approved, error: %v", err)
	}

	// Grant the privilege if the issue is approved.
	if approved && issue.Type == api.IssueGrantRequest {
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &issue.Project.ResourceID})
		if err != nil {
			return nil, err
		}
		var newConditionExpr string
		if payload.GrantRequest.Condition != nil {
			newConditionExpr = payload.GrantRequest.Condition.Expression
		}
		updated := false

		userID, err := strconv.Atoi(strings.TrimPrefix(payload.GrantRequest.User, "users/"))
		if err != nil {
			return nil, err
		}
		newUser, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if newUser == nil {
			return nil, status.Errorf(codes.Internal, "user %v not found", userID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Role(payload.GrantRequest.Role) {
				continue
			}
			var oldConditionExpr string
			if binding.Condition != nil {
				oldConditionExpr = binding.Condition.Expression
			}
			if oldConditionExpr != newConditionExpr {
				continue
			}
			// Append
			binding.Members = append(binding.Members, newUser)
			updated = true
			break
		}
		if !updated {
			role := api.Role(strings.TrimPrefix(payload.GrantRequest.Role, "roles/"))
			condition := payload.GrantRequest.Condition
			condition.Description = fmt.Sprintf("#%d", issue.UID)
			policy.Bindings = append(policy.Bindings, &store.PolicyBinding{
				Role:      role,
				Members:   []*store.UserMessage{newUser},
				Condition: condition,
			})
		}
		if _, err := s.store.SetProjectIAMPolicy(ctx, policy, api.SystemBotID, issue.Project.UID); err != nil {
			return nil, err
		}
	}

	stepsSkipped, err := utils.SkipApprovalStepIfNeeded(ctx, s.store, issue.Project.UID, payload.Approval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to skip approval step if needed, error: %v", err)
	}

	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal issue payload, error: %v", err)
	}
	payloadStr := string(payloadBytes)

	issue, err = s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		Payload: &payloadStr,
	}, api.SystemBotID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
	}

	// It's ok to fail to create activity.
	if err := func() error {
		activityPayload, err := protojson.Marshal(&storepb.ActivityIssueCommentCreatePayload{
			Event: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_{
				ApprovalEvent: &storepb.ActivityIssueCommentCreatePayload_ApprovalEvent{
					Status: storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_APPROVED,
				},
			},
			IssueName: issue.Title,
		})
		if err != nil {
			return err
		}
		create := &api.ActivityCreate{
			CreatorID:   principalID,
			ContainerID: issue.UID,
			Type:        api.ActivityIssueCommentCreate,
			Level:       api.ActivityInfo,
			Comment:     "",
			Payload:     string(activityPayload),
		}
		if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
			return err
		}

		if stepsSkipped > 0 {
			for i := 0; i < stepsSkipped; i++ {
				create := &api.ActivityCreate{
					CreatorID:   api.SystemBotID,
					ContainerID: issue.UID,
					Type:        api.ActivityIssueCommentCreate,
					Level:       api.ActivityInfo,
					Comment:     "",
					Payload:     string(activityPayload),
				}
				if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
					return err
				}
			}
		}

		return nil
	}(); err != nil {
		log.Error("failed to create activity after approving review", zap.Error(err))
	}

	review, err := convertToReview(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to review, error: %v", err)
	}
	return review, nil
}

// UpdateReview updates the review.
// It can only update approval_finding_done to false.
func (s *ReviewService) UpdateReview(ctx context.Context, request *v1pb.UpdateReviewRequest) (*v1pb.Review, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	reviewID, err := getReviewID(request.Review.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &reviewID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue, error: %v", err)
	}
	if issue == nil {
		return nil, status.Errorf(codes.NotFound, "issue %d not found", reviewID)
	}

	patch := &store.UpdateIssueMessage{}
	for _, path := range request.UpdateMask.Paths {
		if path == "approval_finding_done" {
			if request.Review.ApprovalFindingDone {
				return nil, status.Errorf(codes.InvalidArgument, "cannot set approval_finding_done to true")
			}
			payload := &storepb.IssuePayload{}
			if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal issue payload, error: %v", err)
			}
			if payload.Approval == nil {
				return nil, status.Errorf(codes.Internal, "issue payload approval is nil")
			}
			if !payload.Approval.ApprovalFindingDone {
				return nil, status.Errorf(codes.FailedPrecondition, "approval template finding is not done")
			}
			payloadBytes, err := protojson.Marshal(&storepb.IssuePayload{
				Approval: &storepb.IssuePayloadApproval{
					ApprovalFindingDone: false,
				},
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to marshal issue payload, error: %v", err)
			}
			payloadStr := string(payloadBytes)
			patch.Payload = &payloadStr

			if issue.PipelineUID != nil {
				if err := s.taskCheckScheduler.SchedulePipelineTaskCheckReport(ctx, *issue.PipelineUID); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to schedule pipeline task check report, error: %v", err)
				}
			}
		}
	}

	issue, err = s.store.UpdateIssueV2(ctx, issue.UID, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
	}

	s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

	review, err := convertToReview(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to review, error: %v", err)
	}
	return review, nil
}

func canUserApproveStep(step *storepb.ApprovalStep, user *store.UserMessage, policy *store.IAMPolicyMessage) (bool, error) {
	if len(step.Nodes) != 1 {
		return false, errors.Errorf("expecting one node but got %v", len(step.Nodes))
	}
	if step.Type != storepb.ApprovalStep_ANY {
		return false, errors.Errorf("expecting ANY step type but got %v", step.Type)
	}
	node := step.Nodes[0]
	if node.Type != storepb.ApprovalNode_ANY_IN_GROUP {
		return false, errors.Errorf("expecting ANY_IN_GROUP node type but got %v", node.Type)
	}

	userHasProjectRole := map[string]bool{}
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member.ID == user.ID {
				userHasProjectRole[convertToRoleName(string(binding.Role))] = true
				break
			}
		}
	}
	switch val := node.Payload.(type) {
	case *storepb.ApprovalNode_GroupValue_:
		switch val.GroupValue {
		case storepb.ApprovalNode_GROUP_VALUE_UNSPECIFILED:
			return false, errors.Errorf("invalid group value")
		case storepb.ApprovalNode_WORKSPACE_OWNER:
			return user.Role == api.Owner, nil
		case storepb.ApprovalNode_WORKSPACE_DBA:
			return user.Role == api.DBA, nil
		case storepb.ApprovalNode_PROJECT_OWNER:
			return userHasProjectRole[convertToRoleName(string(api.Owner))], nil
		case storepb.ApprovalNode_PROJECT_MEMBER:
			return userHasProjectRole[convertToRoleName(string(api.Developer))], nil
		default:
			return false, errors.Errorf("invalid group value")
		}
	case *storepb.ApprovalNode_Role:
		if userHasProjectRole[val.Role] {
			return true, nil
		}
	default:
		return false, errors.Errorf("invalid node payload type")
	}

	return false, nil
}

func convertToReview(ctx context.Context, store *store.Store, issue *store.IssueMessage) (*v1pb.Review, error) {
	issuePayload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), issuePayload); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal issue payload")
	}

	review := &v1pb.Review{}
	if issuePayload.Approval != nil {
		review.ApprovalFindingDone = issuePayload.Approval.ApprovalFindingDone
		review.ApprovalFindingError = issuePayload.Approval.ApprovalFindingError
		for _, template := range issuePayload.Approval.ApprovalTemplates {
			review.ApprovalTemplates = append(review.ApprovalTemplates, convertToApprovalTemplate(template))
		}
		for _, approver := range issuePayload.Approval.Approvers {
			convertedApprover := &v1pb.Review_Approver{Status: v1pb.Review_Approver_Status(approver.Status)}
			user, err := store.GetUserByID(ctx, int(approver.PrincipalId))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find user by id %v", approver.PrincipalId)
			}
			convertedApprover.Principal = fmt.Sprintf("user:%s", user.Email)

			review.Approvers = append(review.Approvers, convertedApprover)
		}
	}

	return review, nil
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
			GroupValue: v1pb.ApprovalNode_GroupValue(payload.GroupValue),
		}
	case *storepb.ApprovalNode_Role:
		v1node.Payload = &v1pb.ApprovalNode_Role{
			Role: payload.Role,
		}
	}
	return v1node
}
