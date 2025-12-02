package service

import (
	"context"
	"fmt"
	"time"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/common/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ApprovalService is the service for managing approval requests.
type ApprovalService interface {
	// CreateApprovalRequest creates a new approval request
	CreateApprovalRequest(ctx context.Context, req *CreateApprovalRequest) (*v1.ApprovalRequest, error)
	// GetApprovalRequest gets an approval request by ID
	GetApprovalRequest(ctx context.Context, approvalRequestID string) (*v1.ApprovalRequest, error)
	// ListApprovalRequests lists approval requests
	ListApprovalRequests(ctx context.Context, req *ListApprovalRequestsRequest) ([]*v1.ApprovalRequest, error)
	// ProcessApprovalAction processes an approval action (approve/reject)
	ProcessApprovalAction(ctx context.Context, req *ApprovalActionRequest) (*v1.ApprovalRequest, error)
	// CanUserApprove checks if a user can approve a request at the current step
	CanUserApprove(ctx context.Context, userID string, approvalRequestID string) (bool, error)
}

// CreateApprovalRequest is the request for creating an approval request.
type CreateApprovalRequest struct {
	DisplayName    string
	Description    string
	Requester      string
	SensitiveSeverity v1.SensitiveLevel_Severity
	Details        *v1.ApprovalRequestDetails
	ApprovalFlowID string
}

// ListApprovalRequestsRequest is the request for listing approval requests.
type ListApprovalRequestsRequest struct {
	Requester     string
	Approver      string
	Status        v1.ApprovalRequest_Status
	SensitiveSeverity v1.SensitiveLevel_Severity
	PageSize      int32
	PageToken     string
}

// ApprovalActionRequest is the request for processing an approval action.
type ApprovalActionRequest struct {
	ApprovalRequestID string
	Approver          string
	Action            v1.ApprovalLog_Action
	Reason            string
	Comment           string
}

// approvalServiceImpl is the implementation of ApprovalService.
type approvalServiceImpl struct {
	sensitiveApprovalService v1.SensitiveApprovalServiceClient
	store *store.Store
	notificationService NotificationService
}

// NewApprovalService creates a new ApprovalService.
func NewApprovalService(
	sensitiveApprovalService v1.SensitiveApprovalServiceClient,
	store *store.Store,
	notificationService NotificationService,
) ApprovalService {
	return &approvalServiceImpl{
		sensitiveApprovalService: sensitiveApprovalService,
		store: store,
		notificationService: notificationService,
	}
}

// CreateApprovalRequest creates a new approval request.
func (s *approvalServiceImpl) CreateApprovalRequest(ctx context.Context, req *CreateApprovalRequest) (*v1.ApprovalRequest, error) {
	log.Infof("Creating approval request: %s", req.DisplayName)

	// Generate unique ID
	approvalRequestID := uuid.New().String()
	now := timestamppb.Now()

	// Get approval flow if specified
	var approvalFlow *v1.ApprovalFlow
	if req.ApprovalFlowID != "" {
		flowReq := &v1.GetApprovalFlowRequest{
			Name: fmt.Sprintf("approval-flows/%s", req.ApprovalFlowID),
		}
		flowResp, err := s.sensitiveApprovalService.GetApprovalFlow(ctx, flowReq)
		if err != nil {
			log.Errorf("Failed to get approval flow: %v", err)
			return nil, err
		}
		approvalFlow = flowResp
	} else {
		// Find approval flow by sensitive severity
		var err error
		approvalFlow, err = s.getApprovalFlowForSeverity(ctx, req.SensitiveSeverity)
		if err != nil {
			log.Errorf("Failed to find approval flow: %v", err)
			return nil, err
		}
	}

	if approvalFlow == nil {
		log.Infof("No approval flow found for severity %v, auto-approving", req.SensitiveSeverity)
		// Auto-approve if no approval flow is found
		return &v1.ApprovalRequest{
			Name:           fmt.Sprintf("approval-requests/%s", approvalRequestID),
			DisplayName:    req.DisplayName,
			Description:    req.Description,
			Requester:      req.Requester,
			Status:          v1.ApprovalRequest_STATUS_APPROVED,
			SensitiveSeverity: req.SensitiveSeverity,
			CurrentStep:     0,
			CreateTime:      now,
			UpdateTime:      now,
			Details:         req.Details,
		}, nil
	}

	// Create approval request
	approvalRequest := &v1.ApprovalRequest{
		Name:           fmt.Sprintf("approval-requests/%s", approvalRequestID),
		DisplayName:    req.DisplayName,
			Description:    req.Description,
			Requester:      req.Requester,
			Status:          v1.ApprovalRequest_STATUS_PENDING,
			SensitiveSeverity: req.SensitiveSeverity,
			CurrentStep:     1,
			ApprovalFlowName: approvalFlow.Name,
			CreateTime:      now,
			UpdateTime:      now,
			Details:         req.Details,
	}

	// Save to store
	// TODO: Implement store creation

	// Notify approvers
	err := s.notifyApprovers(ctx, approvalRequest, approvalFlow)
	if err != nil {
		log.Warnf("Failed to notify approvers: %v", err)
	}

	log.Infof("Created approval request: %s, Status: %v", approvalRequest.Name, approvalRequest.Status)
	return approvalRequest, nil
}

// GetApprovalRequest gets an approval request by ID.
func (s *approvalServiceImpl) GetApprovalRequest(ctx context.Context, approvalRequestID string) (*v1.ApprovalRequest, error) {
	log.Infof("Getting approval request: %s", approvalRequestID)

	// TODO: Implement store retrieval
	return nil, nil
}

// ListApprovalRequests lists approval requests.
func (s *approvalServiceImpl) ListApprovalRequests(ctx context.Context, req *ListApprovalRequestsRequest) ([]*v1.ApprovalRequest, error) {
	log.Infof("Listing approval requests")

	// TODO: Implement store listing
	return nil, nil
}

// ProcessApprovalAction processes an approval action (approve/reject).
func (s *approvalServiceImpl) ProcessApprovalAction(ctx context.Context, req *ApprovalActionRequest) (*v1.ApprovalRequest, error) {
	log.Infof("Processing approval action: Request=%s, Approver=%s, Action=%v", req.ApprovalRequestID, req.Approver, req.Action)

	// Get approval request
	approvalRequest, err := s.GetApprovalRequest(ctx, req.ApprovalRequestID)
	if err != nil {
		return nil, err
	}

	if approvalRequest == nil {
		return nil, fmt.Errorf("approval request not found: %s", req.ApprovalRequestID)
	}

	// Check if request is already processed
	if approvalRequest.Status != v1.ApprovalRequest_STATUS_PENDING {
		return nil, fmt.Errorf("approval request already %s", approvalRequest.Status.String())
	}

	// Check if user can approve this request
	canApprove, err := s.CanUserApprove(ctx, req.Approver, req.ApprovalRequestID)
	if err != nil {
		return nil, err
	}
	if !canApprove {
		return nil, fmt.Errorf("user %s is not authorized to approve this request", req.Approver)
	}

	// Get approval flow
	approvalFlow, err := s.sensitiveApprovalService.GetApprovalFlow(ctx, &v1.GetApprovalFlowRequest{
		Name: approvalRequest.ApprovalFlowName,
	})
	if err != nil {
		return nil, err
	}

	// Process the action
	err = s.processAction(ctx, approvalRequest, approvalFlow, req)
	if err != nil {
		return nil, err
	}

	return approvalRequest, nil
}

// CanUserApprove checks if a user can approve a request at the current step.
func (s *approvalServiceImpl) CanUserApprove(ctx context.Context, userID string, approvalRequestID string) (bool, error) {
	// TODO: Implement authorization check
	return true, nil
}

// getApprovalFlowForSeverity gets the approval flow for a specific sensitive severity.
func (s *approvalServiceImpl) getApprovalFlowForSeverity(ctx context.Context, severity v1.SensitiveLevel_Severity) (*v1.ApprovalFlow, error) {
	req := &v1.ListApprovalFlowsRequest{}
	resp, err := s.sensitiveApprovalService.ListApprovalFlows(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, flow := range resp.ApprovalFlows {
		if flow.SensitiveSeverity == severity && len(flow.Steps) > 0 {
			return flow, nil
		}
	}

	return nil, nil
}

// processAction processes the approval action.
func (s *approvalServiceImpl) processAction(
	ctx context.Context,
	request *v1.ApprovalRequest,
	flow *v1.ApprovalFlow,
	actionReq *ApprovalActionRequest,
) error {
	// Create approval log
	logEntry := &v1.ApprovalLog{
		ApprovalRequestName: request.Name,
		StepNumber:         request.CurrentStep,
		Approver:           actionReq.Approver,
		Action:             actionReq.Action,
		Reason:             actionReq.Reason,
		Comment:            actionReq.Comment,
		CreateTime:         timestamppb.Now(),
	}

	// Save log
	// TODO: Implement store save

	switch actionReq.Action {
	case v1.ApprovalLog_ACTION_APPROVE:
		// Move to next step or complete
		if request.CurrentStep >= int32(len(flow.Steps)) {
			// All steps approved
			request.Status = v1.ApprovalRequest_STATUS_APPROVED
			request.UpdateTime = timestamppb.Now()
			log.Infof("Approval request %s fully approved", request.Name)
			// Notify requester
			s.notificationService.Notify(ctx, request.Requester, "Approval Request Approved", fmt.Sprintf("Your approval request '%s' has been fully approved", request.DisplayName))
		} else {
			// Move to next step
			request.CurrentStep++
			request.UpdateTime = timestamppb.Now()
			log.Infof("Approval request %s moved to step %d", request.Name, request.CurrentStep)
			// Notify next approvers
			s.notifyStepApprovers(ctx, request, flow, request.CurrentStep)
		}

	case v1.ApprovalLog_ACTION_REJECT:
		// Reject the request
		request.Status = v1.ApprovalRequest_STATUS_REJECTED
		request.UpdateTime = timestamppb.Now()
		log.Infof("Approval request %s rejected", request.Name)
		// Notify requester
		s.notificationService.Notify(ctx, request.Requester, "Approval Request Rejected", fmt.Sprintf("Your approval request '%s' has been rejected: %s", request.DisplayName, actionReq.Reason))

	default:
		return fmt.Errorf("unknown approval action: %v", actionReq.Action)
	}

	// Save updated request
	// TODO: Implement store update

	return nil
}

// notifyApprovers notifies the approvers for the first step.
func (s *approvalServiceImpl) notifyApprovers(ctx context.Context, request *v1.ApprovalRequest, flow *v1.ApprovalFlow) error {
	if len(flow.Steps) == 0 {
		return nil
	}

	return s.notifyStepApprovers(ctx, request, flow, 1)
}

// notifyStepApprovers notifies the approvers for a specific step.
func (s *approvalServiceImpl) notifyStepApprovers(ctx context.Context, request *v1.ApprovalRequest, flow *v1.ApprovalFlow, step int32) error {
	if step < 1 || step > int32(len(flow.Steps)) {
		return fmt.Errorf("invalid step number: %d", step)
	}

	stepConfig := flow.Steps[step-1]
	approvers, err := s.getApproversForStep(ctx, stepConfig)
	if err != nil {
		return err
	}

	for _, approver := range approvers {
		message := fmt.Sprintf("Approval Request #%s: %s\nStep %d/%d: %s\n\nDescription: %s\nSensitivity: %v",
			request.Name,
			request.DisplayName,
			step,
			len(flow.Steps),
			stepConfig.DisplayName,
			request.Description,
			request.SensitiveSeverity.String())

		s.notificationService.Notify(ctx, approver, "Approval Request Pending", message)
	}

	return nil
}

// getApproversForStep gets the list of approvers for a step.
func (s *approvalServiceImpl) getApproversForStep(ctx context.Context, step *v1.ApprovalStep) ([]string, error) {
	var approvers []string

	switch step.ApproverType {
	case v1.ApprovalStep_APPROVER_TYPE_ROLE:
		// Get users by role
		// TODO: Implement role-based user lookup
		approvers = append(approvers, step.Role)

	case v1.ApprovalStep_APPROVER_TYPE_USER:
		approvers = append(approvers, step.Approver)

	case v1.ApprovalStep_APPROVER_TYPE_GROUP:
		// Get users in group
		// TODO: Implement group-based user lookup
		approvers = append(approvers, step.Group)

	default:
		return nil, fmt.Errorf("unknown approver type: %v", step.ApproverType)
	}

	return approvers, nil
}
