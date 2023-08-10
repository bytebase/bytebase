package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

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

// IssueService implements the issue service.
type IssueService struct {
	v1pb.UnimplementedIssueServiceServer
	store              *store.Store
	activityManager    *activity.Manager
	taskScheduler      *taskrun.Scheduler
	taskCheckScheduler *taskcheck.Scheduler
	relayRunner        *relay.Runner
	stateCfg           *state.State
}

// NewIssueService creates a new IssueService.
func NewIssueService(store *store.Store, activityManager *activity.Manager, taskScheduler *taskrun.Scheduler, taskCheckScheduler *taskcheck.Scheduler, relayRunner *relay.Runner, stateCfg *state.State) *IssueService {
	return &IssueService{
		store:              store,
		activityManager:    activityManager,
		taskScheduler:      taskScheduler,
		taskCheckScheduler: taskCheckScheduler,
		relayRunner:        relayRunner,
		stateCfg:           stateCfg,
	}
}

// GetIssue gets a issue.
func (s *IssueService) GetIssue(ctx context.Context, request *v1pb.GetIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if request.Force {
		externalApprovalType := api.ExternalApprovalTypeRelay
		approvals, err := s.store.ListExternalApprovalV2(ctx, &store.ListExternalApprovalMessage{
			Type:     &externalApprovalType,
			IssueUID: &issue.UID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list external approvals, error: %v", err)
		}
		var errs error
		for _, approval := range approvals {
			msg := relay.CheckExternalApprovalChanMessage{
				ExternalApproval: approval,
				ErrChan:          make(chan error, 1),
			}
			s.relayRunner.CheckExternalApprovalChan <- msg
			err := <-msg.ErrChan
			if err != nil {
				err = errors.Wrapf(err, "failed to check external approval status, issueUID %d", approval.IssueUID)
				errs = multierr.Append(errs, err)
			}
		}
		if errs != nil {
			return nil, status.Errorf(codes.Internal, "failed to check external approval status, error: %v", errs)
		}
		issue, err = s.getIssueMessage(ctx, request.Name)
		if err != nil {
			return nil, err
		}
	}
	issueV1, err := convertToIssue(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// CreateIssue creates a issue.
func (s *IssueService) CreateIssue(ctx context.Context, request *v1pb.CreateIssueRequest) (*v1pb.Issue, error) {
	creatorID := ctx.Value(common.PrincipalIDContextKey).(int)
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project not found for id: %v", projectID)
	}

	switch request.Issue.Type {
	case v1pb.Issue_TYPE_UNSPECIFIED:
		return nil, status.Errorf(codes.InvalidArgument, "issue type is required")
	case v1pb.Issue_GRANT_REQUEST:
		return nil, status.Errorf(codes.Unimplemented, "issue type %q is not implemented yet", request.Issue.Type)
	case v1pb.Issue_DATABASE_CHANGE:
		// TODO(p0ny): refactor
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown issue type %q", request.Issue.Type)
	}

	if request.Issue.Plan == "" {
		return nil, status.Errorf(codes.InvalidArgument, "plan is required")
	}

	var planUID *int64
	planID, err := common.GetPlanID(request.Issue.Plan)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}
	planUID = &plan.UID

	assigneeEmail, err := common.GetUserEmail(request.Issue.Assignee)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	assignee, err := s.store.GetUser(ctx, &store.FindUserMessage{Email: &assigneeEmail})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user by email %q, error: %v", assigneeEmail, err)
	}
	if assignee == nil {
		return nil, status.Errorf(codes.NotFound, "assignee not found for email: %q", assigneeEmail)
	}

	issueCreateMessage := &store.IssueMessage{
		Project:       project,
		PlanUID:       planUID,
		PipelineUID:   nil,
		Title:         request.Issue.Title,
		Status:        api.IssueOpen,
		Type:          api.IssueDatabaseGeneral,
		Description:   request.Issue.Description,
		Assignee:      assignee,
		NeedAttention: false,
	}

	// TODO(p0ny): find approval template
	issueCreatePayload := &storepb.IssuePayload{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: true,
			ApprovalTemplates:   nil,
			Approvers:           nil,
		},
	}
	issueCreatePayloadBytes, err := protojson.Marshal(issueCreatePayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal issue payload, error: %v", err)
	}
	issueCreateMessage.Payload = string(issueCreatePayloadBytes)

	issue, err := s.store.CreateIssueV2(ctx, issueCreateMessage, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create issue, error: %v", err)
	}
	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Title,
	}
	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}
	activityCreate := &store.ActivityMessage{
		CreatorUID:   creatorID,
		ContainerUID: issue.UID,
		Type:         api.ActivityIssueCreate,
		Level:        api.ActivityInfo,
		Payload:      string(bytes),
	}
	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: issue,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}

	converted, err := convertToIssue(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}

	return converted, nil
}

// ApproveIssue approves the approval flow of the issue.
func (s *IssueService) ApproveIssue(ctx context.Context, request *v1pb.ApproveIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
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
		return nil, status.Errorf(codes.InvalidArgument, "cannot approve because the issue has been rejected")
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return nil, status.Errorf(codes.InvalidArgument, "the issue has been approved")
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
		if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, s.store, issue, payload.GrantRequest); err != nil {
			return nil, err
		}
		userID, err := strconv.Atoi(strings.TrimPrefix(payload.GrantRequest.User, "users/"))
		if err != nil {
			return nil, err
		}
		newUser, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		// Post project IAM policy update activity.
		if _, err := s.activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:   api.SystemBotID,
			ContainerUID: issue.Project.UID,
			Type:         api.ActivityProjectMemberCreate,
			Level:        api.ActivityInfo,
			Comment:      fmt.Sprintf("Granted %s to %s (%s).", newUser.Name, newUser.Email, payload.GrantRequest.Role),
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
		log.Error("failed to create skipping steps activity after approving issue", zap.Error(err))
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
		log.Error("failed to create approval step pending activity after creating issue", zap.Error(err))
	}

	s.onIssueApproved(ctx, issue)

	issueV1, err := convertToIssue(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// RejectIssue rejects a issue.
func (s *IssueService) RejectIssue(ctx context.Context, request *v1pb.RejectIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
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
		return nil, status.Errorf(codes.InvalidArgument, "cannot reject because the issue has been rejected")
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return nil, status.Errorf(codes.InvalidArgument, "the issue has been approved")
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
		log.Error("failed to create activity after rejecting issue", zap.Error(err))
	}

	issueV1, err := convertToIssue(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// RequestIssue requests a issue.
func (s *IssueService) RequestIssue(ctx context.Context, request *v1pb.RequestIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
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
		return nil, status.Errorf(codes.InvalidArgument, "cannot request issues because the issue is not rejected")
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user by id %v", principalID)
	}

	canRequest := canRequestIssue(issue.Creator, user)
	if !canRequest {
		return nil, status.Errorf(codes.PermissionDenied, "cannot request issues because you are not the issue creator")
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
		log.Error("failed to create skipping steps activity after approving issue", zap.Error(err))
	}

	issueV1, err := convertToIssue(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// UpdateIssue updates the issue.
// It can only update approval_finding_done to false.
func (s *IssueService) UpdateIssue(ctx context.Context, request *v1pb.UpdateIssueRequest) (*v1pb.Issue, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	issue, err := s.getIssueMessage(ctx, request.Issue.Name)
	if err != nil {
		return nil, err
	}

	updateMasks := map[string]bool{}

	patch := &store.UpdateIssueMessage{}
	for _, path := range request.UpdateMask.Paths {
		updateMasks[path] = true
		switch path {
		case "approval_finding_done":
			if request.Issue.ApprovalFindingDone {
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

		case "title":
			patch.Title = &request.Issue.Title

		case "description":
			patch.Description = &request.Issue.Description

		case "subscribers":
			var subscribers []*store.UserMessage
			for _, subscriber := range request.Issue.Subscribers {
				subscriberEmail, err := common.GetUserEmail(subscriber)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "failed to get user email from %v, error: %v", subscriber, err)
				}
				user, err := s.store.GetUser(ctx, &store.FindUserMessage{Email: &subscriberEmail})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get user %v, error: %v", subscriberEmail, err)
				}
				if user == nil {
					return nil, status.Errorf(codes.NotFound, "user %v not found", subscriber)
				}
				subscribers = append(subscribers, user)
			}
			patch.Subscribers = &subscribers

		case "assignee":
			assigneeEmail, err := common.GetUserEmail(request.Issue.Assignee)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to get user email from %v, error: %v", request.Issue.Assignee, err)
			}
			user, err := s.store.GetUser(ctx, &store.FindUserMessage{Email: &assigneeEmail})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get user %v, error: %v", assigneeEmail, err)
			}
			if user == nil {
				return nil, status.Errorf(codes.NotFound, "user %v not found", request.Issue.Assignee)
			}
			patch.Assignee = user
		}
	}

	issue, err = s.store.UpdateIssueV2(ctx, issue.UID, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
	}

	if updateMasks["approval_finding_done"] {
		s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
	}

	issueV1, err := convertToIssue(ctx, s.store, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// CreateIssueComment creates the issue comment.
func (s *IssueService) CreateIssueComment(ctx context.Context, request *v1pb.CreateIssueCommentRequest) (*v1pb.IssueComment, error) {
	if request.IssueComment.Comment == "" {
		return nil, status.Errorf(codes.InvalidArgument, "issue comment is empty")
	}
	issue, err := s.getIssueMessage(ctx, request.Parent)
	if err != nil {
		return nil, err
	}

	// TODO: migrate to store v2
	activityCreate := &store.ActivityMessage{
		CreatorUID:   ctx.Value(common.PrincipalIDContextKey).(int),
		ContainerUID: issue.UID,
		Type:         api.ActivityIssueCommentCreate,
		Level:        api.ActivityInfo,
		Comment:      request.IssueComment.Comment,
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
		return nil, status.Errorf(codes.Internal, "failed to create issue comment: %v", err.Error())
	}
	return &v1pb.IssueComment{
		Uid:        fmt.Sprintf("%d", activity.UID),
		Comment:    activity.Comment,
		Payload:    activity.Payload,
		CreateTime: timestamppb.New(time.Unix(activity.CreatedTs, 0)),
		UpdateTime: timestamppb.New(time.Unix(activity.UpdatedTs, 0)),
	}, nil
}

// UpdateIssueComment updates the issue comment.
func (s *IssueService) UpdateIssueComment(ctx context.Context, request *v1pb.UpdateIssueCommentRequest) (*v1pb.IssueComment, error) {
	if request.UpdateMask.Paths == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}
	activityUID, err := strconv.Atoi(request.IssueComment.Uid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, `invalid comment id "%s": %v`, request.IssueComment.Uid, err.Error())
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
			update.Comment = &request.IssueComment.Comment
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask: "%s"`, path)
		}
	}

	activity, err := s.store.UpdateActivityV2(ctx, update)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, status.Errorf(codes.NotFound, "cannot found the issue comment %s", request.IssueComment.Uid)
		}
		return nil, status.Errorf(codes.Internal, "failed to update the issue comment with error: %v", err.Error())
	}

	return &v1pb.IssueComment{
		Uid:        fmt.Sprintf("%d", activity.UID),
		Comment:    activity.Comment,
		Payload:    activity.Payload,
		CreateTime: timestamppb.New(time.Unix(activity.CreatedTs, 0)),
		UpdateTime: timestamppb.New(time.Unix(activity.UpdatedTs, 0)),
	}, nil
}

func (s *IssueService) onIssueApproved(ctx context.Context, issue *store.IssueMessage) {
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

func (s *IssueService) getIssueMessage(ctx context.Context, name string) (*store.IssueMessage, error) {
	issueID, err := common.GetIssueID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issueID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue, error: %v", err)
	}
	if issue == nil {
		return nil, status.Errorf(codes.NotFound, "issue %d not found", issueID)
	}
	return issue, nil
}

func canRequestIssue(issueCreator *store.UserMessage, user *store.UserMessage) bool {
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

func convertToIssue(ctx context.Context, s *store.Store, issue *store.IssueMessage) (*v1pb.Issue, error) {
	issuePayload := &storepb.IssuePayload{}
	if err := protojson.Unmarshal([]byte(issue.Payload), issuePayload); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal issue payload")
	}

	issueV1 := &v1pb.Issue{
		Name:                 fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, issue.Project.ResourceID, common.IssuePrefix, issue.UID),
		Uid:                  fmt.Sprintf("%d", issue.UID),
		Title:                issue.Title,
		Description:          issue.Description,
		Type:                 convertToIssueType(issue.Type),
		Status:               convertToIssueStatus(issue.Status),
		Assignee:             fmt.Sprintf("%s%s", common.UserNamePrefix, issue.Assignee.Email),
		AssigneeAttention:    issue.NeedAttention,
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
	}

	if issue.PlanUID != nil {
		issueV1.Plan = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, issue.Project.ResourceID, common.PlanPrefix, *issue.PlanUID)
	}
	if issue.PipelineUID != nil {
		issueV1.Rollout = fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, issue.Project.ResourceID, common.RolloutPrefix, *issue.PipelineUID)
	}

	for _, subscriber := range issue.Subscribers {
		issueV1.Subscribers = append(issueV1.Subscribers, fmt.Sprintf("%s%s", common.UserNamePrefix, subscriber.Email))
	}

	if issuePayload.Approval != nil {
		issueV1.ApprovalFindingDone = issuePayload.Approval.ApprovalFindingDone
		issueV1.ApprovalFindingError = issuePayload.Approval.ApprovalFindingError
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

func convertToIssueType(t api.IssueType) v1pb.Issue_Type {
	switch t {
	case api.IssueDatabaseCreate, api.IssueDatabaseSchemaUpdate, api.IssueDatabaseSchemaUpdateGhost, api.IssueDatabaseDataUpdate, api.IssueDatabaseRestorePITR, api.IssueDatabaseGeneral:
		return v1pb.Issue_DATABASE_CHANGE
	case api.IssueGrantRequest:
		return v1pb.Issue_GRANT_REQUEST
	case api.IssueGeneral:
		return v1pb.Issue_TYPE_UNSPECIFIED
	default:
		return v1pb.Issue_TYPE_UNSPECIFIED
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
