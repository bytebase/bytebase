package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/runner/relay"
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
	relayRunner        *relay.Runner
	stateCfg           *state.State
}

// NewReviewService creates a new ReviewService.
func NewReviewService(store *store.Store, activityManager *activity.Manager, taskScheduler *taskrun.Scheduler, taskCheckScheduler *taskcheck.Scheduler, relayRunner *relay.Runner, stateCfg *state.State) *ReviewService {
	return &ReviewService{
		store:              store,
		activityManager:    activityManager,
		taskScheduler:      taskScheduler,
		taskCheckScheduler: taskCheckScheduler,
		relayRunner:        relayRunner,
		stateCfg:           stateCfg,
	}
}

// GetReview gets a review.
// Currently, only review.ApprovalTemplates and review.Approvers are set.
func (s *ReviewService) GetReview(ctx context.Context, request *v1pb.GetReviewRequest) (*v1pb.Review, error) {
	issue, err := s.getIssue(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	review, err := convertToReview(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to review, error: %v", err)
	}
	return review, nil
}

// ApproveReview approves the approval flow of the review.
func (s *ReviewService) ApproveReview(ctx context.Context, request *v1pb.ApproveReviewRequest) (*v1pb.Review, error) {
	issue, err := s.getIssue(ctx, request.Name)
	if err != nil {
		return nil, err
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

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot approve because the review has been rejected")
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

	canApprove, err := isUserReviewer(step, user, policy)
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

	newApprovers, activityCreates, err := utils.HandleIncomingApprovalSteps(ctx, s.store, s.relayRunner.Client, issue, payload.Approval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle incoming approval steps, error: %v", err)
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, newApprovers...)
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
		role := api.Role(strings.TrimPrefix(payload.GrantRequest.Role, "roles/"))
		if !updated {
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
		// Post project IAM policy update activity.
		if _, err := s.activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.Project.UID,
			Type:         api.ActivityProjectMemberCreate,
			Level:        api.ActivityInfo,
			Comment:      fmt.Sprintf("Granted %s to %s (%s).", newUser.Name, newUser.Email, role),
		}, &activity.Metadata{}); err != nil {
			log.Warn("Failed to create project activity", zap.Error(err))
		}
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
		create := &store.ActivityMessage{
			CreatorUID:   principalID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueCommentCreate,
			Level:        api.ActivityInfo,
			Comment:      request.Comment,
			Payload:      string(activityPayload),
		}
		if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
			return err
		}

		for _, create := range activityCreates {
			if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
				return err
			}
		}

		return nil
	}(); err != nil {
		log.Error("failed to create skipping steps activity after approving review", zap.Error(err))
	}

	if err := func() error {
		if len(payload.Approval.ApprovalTemplates) != 1 {
			return nil
		}
		approvalStep := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
		if approvalStep == nil {
			return nil
		}
		protoPayload, err := protojson.Marshal(&storepb.ActivityIssueApprovalNotifyPayload{
			ApprovalStep: approvalStep,
		})
		if err != nil {
			return err
		}
		activityPayload, err := json.Marshal(api.ActivityIssueApprovalNotifyPayload{
			ProtoPayload: string(protoPayload),
		})
		if err != nil {
			return err
		}

		create := &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueApprovalNotify,
			Level:        api.ActivityInfo,
			Comment:      "",
			Payload:      string(activityPayload),
		}
		if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{Issue: issue}); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		log.Error("failed to create approval step pending activity after creating review", zap.Error(err))
	}

	s.onReviewApproved(ctx, issue)

	review, err := convertToReview(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to review, error: %v", err)
	}
	return review, nil
}

// RejectReview rejects a review.
func (s *ReviewService) RejectReview(ctx context.Context, request *v1pb.RejectReviewRequest) (*v1pb.Review, error) {
	issue, err := s.getIssue(ctx, request.Name)
	if err != nil {
		return nil, err
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

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot reject because the review has been rejected")
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

	canApprove, err := isUserReviewer(step, user, policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if principal can reject step, error: %v", err)
	}
	if !canApprove {
		return nil, status.Errorf(codes.PermissionDenied, "cannot reject because the user does not have the required permission")
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:      storepb.IssuePayloadApproval_Approver_REJECTED,
		PrincipalId: int32(principalID),
	})

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
					Status: storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_REJECTED,
				},
			},
			IssueName: issue.Title,
		})
		if err != nil {
			return err
		}
		create := &store.ActivityMessage{
			CreatorUID:   principalID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueCommentCreate,
			Level:        api.ActivityInfo,
			Comment:      request.Comment,
			Payload:      string(activityPayload),
		}
		if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		log.Error("failed to create activity after rejecting review", zap.Error(err))
	}

	review, err := convertToReview(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to review, error: %v", err)
	}
	return review, nil
}

// RequestReview requests a review.
func (s *ReviewService) RequestReview(ctx context.Context, request *v1pb.RequestReviewRequest) (*v1pb.Review, error) {
	issue, err := s.getIssue(ctx, request.Name)
	if err != nil {
		return nil, err
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

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep == nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot request reviews because the issue is not rejected")
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user by id %v", principalID)
	}

	canRequest := canRequestReview(issue.Creator, user)
	if !canRequest {
		return nil, status.Errorf(codes.PermissionDenied, "cannot request reviews because you are not the issue creator")
	}

	var newApprovers []*storepb.IssuePayloadApproval_Approver
	for _, approver := range payload.Approval.Approvers {
		if approver.Status == storepb.IssuePayloadApproval_Approver_REJECTED {
			continue
		}
		newApprovers = append(newApprovers, approver)
	}
	payload.Approval.Approvers = newApprovers

	newApprovers, activityCreates, err := utils.HandleIncomingApprovalSteps(ctx, s.store, s.relayRunner.Client, issue, payload.Approval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle incoming approval steps, error: %v", err)
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, newApprovers...)
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
					Status: storepb.ActivityIssueCommentCreatePayload_ApprovalEvent_PENDING,
				},
			},
			IssueName: issue.Title,
		})
		if err != nil {
			return err
		}
		create := &store.ActivityMessage{
			CreatorUID:   principalID,
			ContainerUID: issue.UID,
			Type:         api.ActivityIssueCommentCreate,
			Level:        api.ActivityInfo,
			Comment:      request.Comment,
			Payload:      string(activityPayload),
		}
		if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
			return err
		}

		for _, create := range activityCreates {
			if _, err := s.activityManager.CreateActivity(ctx, create, &activity.Metadata{}); err != nil {
				return err
			}
		}

		return nil
	}(); err != nil {
		log.Error("failed to create skipping steps activity after approving review", zap.Error(err))
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
	issue, err := s.getIssue(ctx, request.Review.Name)
	if err != nil {
		return nil, err
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

// CreateReviewComment creates the review comment.
func (s *ReviewService) CreateReviewComment(ctx context.Context, request *v1pb.CreateReviewCommentRequest) (*v1pb.ReviewComment, error) {
	if request.ReviewComment.Comment == "" {
		return nil, status.Errorf(codes.InvalidArgument, "review comment is empty")
	}
	issue, err := s.getIssue(ctx, request.Parent)
	if err != nil {
		return nil, err
	}

	// TODO: migrate to store v2
	activityCreate := &store.ActivityMessage{
		CreatorUID:   ctx.Value(common.PrincipalIDContextKey).(int),
		ContainerUID: issue.UID,
		Type:         api.ActivityIssueCommentCreate,
		Level:        api.ActivityInfo,
		Comment:      request.ReviewComment.Comment,
	}

	var payload api.ActivityIssueCommentCreatePayload
	if activityCreate.Payload != "" {
		if err := json.Unmarshal([]byte(activityCreate.Payload), &payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal payload: %v", err.Error())
		}
	}
	payload.IssueName = issue.Title
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal payload: %v", err.Error())
	}
	activityCreate.Payload = string(bytes)

	activity, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create review comment: %v", err.Error())
	}
	return &v1pb.ReviewComment{
		Uid:        fmt.Sprintf("%d", activity.UID),
		Comment:    activity.Comment,
		Payload:    activity.Payload,
		CreateTime: timestamppb.New(time.Unix(activity.CreatedTs, 0)),
		UpdateTime: timestamppb.New(time.Unix(activity.UpdatedTs, 0)),
	}, nil
}

// UpdateReviewComment updates the review comment.
func (s *ReviewService) UpdateReviewComment(ctx context.Context, request *v1pb.UpdateReviewCommentRequest) (*v1pb.ReviewComment, error) {
	if request.UpdateMask.Paths == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}
	activityUID, err := strconv.Atoi(request.ReviewComment.Uid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, `invalid comment id "%s": %v`, request.ReviewComment.Uid, err.Error())
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	update := &store.UpdateActivityMessage{
		UID:        activityUID,
		CreatorUID: &principalID,
		UpdaterUID: principalID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "comment":
			update.Comment = &request.ReviewComment.Comment
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask: "%s"`, path)
		}
	}

	activity, err := s.store.UpdateActivityV2(ctx, update)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, status.Errorf(codes.NotFound, "cannot found the review comment %s", request.ReviewComment.Uid)
		}
		return nil, status.Errorf(codes.Internal, "failed to update the review comment with error: %v", err.Error())
	}

	return &v1pb.ReviewComment{
		Uid:        fmt.Sprintf("%d", activity.UID),
		Comment:    activity.Comment,
		Payload:    activity.Payload,
		CreateTime: timestamppb.New(time.Unix(activity.CreatedTs, 0)),
		UpdateTime: timestamppb.New(time.Unix(activity.UpdatedTs, 0)),
	}, nil
}

func (s *ReviewService) onReviewApproved(ctx context.Context, issue *store.IssueMessage) {
	if issue.Type == api.IssueGrantRequest {
		if err := func() error {
			payload := &storepb.IssuePayload{}
			if err := protojson.Unmarshal([]byte(issue.Payload), payload); err != nil {
				return errors.Wrap(err, "failed to unmarshal issue payload")
			}
			approved, err := utils.CheckApprovalApproved(payload.Approval)
			if err != nil {
				return errors.Wrap(err, "failed to check if the approval is approved")
			}
			if approved {
				if err := s.taskScheduler.ChangeIssueStatus(ctx, issue, api.IssueDone, api.SystemBotID, ""); err != nil {
					return errors.Wrap(err, "failed to update issue status")
				}
			}
			return nil
		}(); err != nil {
			log.Debug("failed to update issue status to done if grant request issue is approved", zap.Error(err))
		}
	}
}

func (s *ReviewService) getIssue(ctx context.Context, name string) (*store.IssueMessage, error) {
	reviewID, err := getReviewID(name)
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
	return issue, nil
}

func canRequestReview(issueCreator *store.UserMessage, user *store.UserMessage) bool {
	return issueCreator.ID == user.ID
}

func isUserReviewer(step *storepb.ApprovalStep, user *store.UserMessage, policy *store.IAMPolicyMessage) (bool, error) {
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
	case *storepb.ApprovalNode_ExternalNodeId:
		return true, nil
	default:
		return false, errors.Errorf("invalid node payload type")
	}

	return false, nil
}

func convertToReview(ctx context.Context, s *store.Store, issue *store.IssueMessage) (*v1pb.Review, error) {
	issuePayload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), issuePayload); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal issue payload")
	}

	review := &v1pb.Review{
		Name:              fmt.Sprintf("%s%s/%s%d", projectNamePrefix, issue.Project.ResourceID, reviewPrefix, issue.UID),
		Uid:               fmt.Sprintf("%d", issue.UID),
		Title:             issue.Title,
		Description:       issue.Description,
		Status:            convertToReviewStatus(issue.Status),
		Assignee:          fmt.Sprintf("%s%s", userNamePrefix, issue.Assignee.Email),
		AssigneeAttention: issue.NeedAttention,
		Creator:           fmt.Sprintf("%s%s", userNamePrefix, issue.Creator.Email),
		CreateTime:        timestamppb.New(issue.CreatedTime),
		UpdateTime:        timestamppb.New(issue.UpdatedTime),
	}

	for _, subscriber := range issue.Subscribers {
		review.Subscribers = append(review.Subscribers, fmt.Sprintf("%s%s", userNamePrefix, subscriber.Email))
	}

	switch issue.Status {
	case api.IssueOpen:
		review.Status = v1pb.ReviewStatus_OPEN
	case api.IssueDone:
		review.Status = v1pb.ReviewStatus_DONE
	case api.IssueCanceled:
		review.Status = v1pb.ReviewStatus_CANCELED
	default:
		review.Status = v1pb.ReviewStatus_REVIEW_STATUS_UNSPECIFIED
	}

	if issuePayload.Approval != nil {
		review.ApprovalFindingDone = issuePayload.Approval.ApprovalFindingDone
		review.ApprovalFindingError = issuePayload.Approval.ApprovalFindingError
		for _, template := range issuePayload.Approval.ApprovalTemplates {
			review.ApprovalTemplates = append(review.ApprovalTemplates, convertToApprovalTemplate(template))
		}
		for _, approver := range issuePayload.Approval.Approvers {
			convertedApprover := &v1pb.Review_Approver{Status: v1pb.Review_Approver_Status(approver.Status)}
			user, err := s.GetUserByID(ctx, int(approver.PrincipalId))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find user by id %v", approver.PrincipalId)
			}
			convertedApprover.Principal = fmt.Sprintf("users/%s", user.Email)
			review.Approvers = append(review.Approvers, convertedApprover)
		}
	}

	return review, nil
}

func convertToReviewStatus(status api.IssueStatus) v1pb.ReviewStatus {
	switch status {
	case api.IssueOpen:
		return v1pb.ReviewStatus_OPEN
	case api.IssueDone:
		return v1pb.ReviewStatus_DONE
	case api.IssueCanceled:
		return v1pb.ReviewStatus_CANCELED
	default:
		return v1pb.ReviewStatus_REVIEW_STATUS_UNSPECIFIED
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
