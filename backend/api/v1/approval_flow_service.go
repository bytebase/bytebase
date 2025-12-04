package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// ApprovalFlowService implements the v1.ApprovalFlowService interface.
type ApprovalFlowService struct {
	v1pb.UnimplementedApprovalFlowServiceServer

	store *store.Store
}

// NewApprovalFlowService creates a new ApprovalFlowService.
func NewApprovalFlowService(store *store.Store) *ApprovalFlowService {
	return &ApprovalFlowService{
		store: store,
	}
}

// ListApprovalFlows implements v1.ApprovalFlowServiceServer.ListApprovalFlows.
func (s *ApprovalFlowService) ListApprovalFlows(ctx context.Context, req *v1pb.ListApprovalFlowsRequest) (*v1pb.ListApprovalFlowsResponse, error) {
	find := &store.FindApprovalFlowMessage{}
	if req.WorkspaceId != 0 {
		find.WorkspaceID = &req.WorkspaceId
	}
	if req.Name != "" {
		find.Name = &req.Name
	}
	if req.Enabled {
		find.Enabled = &req.Enabled
	}

	flows, err := s.store.ListApprovalFlows(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list approval flows: %v", err)
	}

	return &v1pb.ListApprovalFlowsResponse{
		Flows:        flows,
		NextPageToken: "", // TODO: Implement pagination
		TotalSize:     int32(len(flows)),
	}, nil
}

// GetApprovalFlow implements v1.ApprovalFlowServiceServer.GetApprovalFlow.
func (s *ApprovalFlowService) GetApprovalFlow(ctx context.Context, req *v1pb.GetApprovalFlowRequest) (*v1pb.GetApprovalFlowResponse, error) {
	flow, err := s.store.GetApprovalFlow(ctx, req.FlowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get approval flow: %v", err)
	}
	if flow == nil {
		return nil, status.Errorf(codes.NotFound, "approval flow not found")
	}

	return &v1pb.GetApprovalFlowResponse{
		Flow: flow,
	}, nil
}

// CreateApprovalFlow implements v1.ApprovalFlowServiceServer.CreateApprovalFlow.
func (s *ApprovalFlowService) CreateApprovalFlow(ctx context.Context, req *v1pb.CreateApprovalFlowRequest) (*v1pb.CreateApprovalFlowResponse, error) {
	if req.Flow == nil {
		return nil, status.Errorf(codes.InvalidArgument, "flow is required")
	}

	// Set default values
	if req.Flow.Enabled == false {
		req.Flow.Enabled = true
	}

	flow, err := s.store.CreateApprovalFlow(ctx, req.Flow)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create approval flow: %v", err)
	}

	return &v1pb.CreateApprovalFlowResponse{
		Flow: flow,
	}, nil
}

// UpdateApprovalFlow implements v1.ApprovalFlowServiceServer.UpdateApprovalFlow.
func (s *ApprovalFlowService) UpdateApprovalFlow(ctx context.Context, req *v1pb.UpdateApprovalFlowRequest) (*v1pb.UpdateApprovalFlowResponse, error) {
	if req.Flow == nil {
		return nil, status.Errorf(codes.InvalidArgument, "flow is required")
	}

	// Ensure the flow ID matches
	req.Flow.Id = req.FlowId

	flow, err := s.store.UpdateApprovalFlow(ctx, req.Flow)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update approval flow: %v", err)
	}

	return &v1pb.UpdateApprovalFlowResponse{
		Flow: flow,
	}, nil
}

// DeleteApprovalFlow implements v1.ApprovalFlowServiceServer.DeleteApprovalFlow.
func (s *ApprovalFlowService) DeleteApprovalFlow(ctx context.Context, req *v1pb.DeleteApprovalFlowRequest) (*emptypb.Empty, error) {
	err := s.store.DeleteApprovalFlow(ctx, req.FlowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete approval flow: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListApprovalTemplates implements v1.ApprovalFlowServiceServer.ListApprovalTemplates.
func (s *ApprovalFlowService) ListApprovalTemplates(ctx context.Context, req *v1pb.ListApprovalTemplatesRequest) (*v1pb.ListApprovalTemplatesResponse, error) {
	find := &store.FindApprovalTemplateMessage{}
	if req.WorkspaceId != 0 {
		find.WorkspaceID = &req.WorkspaceId
	}
	if req.SensitiveLevel != storepb.SensitiveDataLevel_SENSITIVE_DATA_LEVEL_UNSPECIFIED {
		find.SensitiveLevel = &req.SensitiveLevel
	}
	if req.Enabled {
		find.Enabled = &req.Enabled
	}

	templates, err := s.store.ListApprovalTemplates(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list approval templates: %v", err)
	}

	return &v1pb.ListApprovalTemplatesResponse{
		Templates:    templates,
		NextPageToken: "", // TODO: Implement pagination
		TotalSize:     int32(len(templates)),
	}, nil
}

// GetApprovalTemplate implements v1.ApprovalFlowServiceServer.GetApprovalTemplate.
func (s *ApprovalFlowService) GetApprovalTemplate(ctx context.Context, req *v1pb.GetApprovalTemplateRequest) (*v1pb.GetApprovalTemplateResponse, error) {
	template, err := s.store.GetApprovalTemplate(ctx, req.TemplateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get approval template: %v", err)
	}
	if template == nil {
		return nil, status.Errorf(codes.NotFound, "approval template not found")
	}

	return &v1pb.GetApprovalTemplateResponse{
		Template: template,
	}, nil
}

// CreateApprovalTemplate implements v1.ApprovalFlowServiceServer.CreateApprovalTemplate.
func (s *ApprovalFlowService) CreateApprovalTemplate(ctx context.Context, req *v1pb.CreateApprovalTemplateRequest) (*v1pb.CreateApprovalTemplateResponse, error) {
	if req.Template == nil {
		return nil, status.Errorf(codes.InvalidArgument, "template is required")
	}

	// Set default values
	if req.Template.Enabled == false {
		req.Template.Enabled = true
	}

	template, err := s.store.CreateApprovalTemplate(ctx, req.Template)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create approval template: %v", err)
	}

	return &v1pb.CreateApprovalTemplateResponse{
		Template: template,
	}, nil
}

// UpdateApprovalTemplate implements v1.ApprovalFlowServiceServer.UpdateApprovalTemplate.
func (s *ApprovalFlowService) UpdateApprovalTemplate(ctx context.Context, req *v1pb.UpdateApprovalTemplateRequest) (*v1pb.UpdateApprovalTemplateResponse, error) {
	if req.Template == nil {
		return nil, status.Errorf(codes.InvalidArgument, "template is required")
	}

	// Ensure the template ID matches
	req.Template.Id = req.TemplateId

	template, err := s.store.UpdateApprovalTemplate(ctx, req.Template)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update approval template: %v", err)
	}

	return &v1pb.UpdateApprovalTemplateResponse{
		Template: template,
	}, nil
}

// DeleteApprovalTemplate implements v1.ApprovalFlowServiceServer.DeleteApprovalTemplate.
func (s *ApprovalFlowService) DeleteApprovalTemplate(ctx context.Context, req *v1pb.DeleteApprovalTemplateRequest) (*emptypb.Empty, error) {
	err := s.store.DeleteApprovalTemplate(ctx, req.TemplateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete approval template: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// Approve implements v1.ApprovalFlowServiceServer.Approve.
func (s *ApprovalFlowService) Approve(ctx context.Context, req *v1pb.ApproveRequest) (*v1pb.ApproveResponse, error) {
	if req.ApprovalId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "approval_id is required")
	}

	// TODO: Implement approval logic
	// 1. Get the approval history record
	// 2. Verify the current user has permission to approve
	// 3. Update the approval status to APPROVED
	// 4. Check if all approvals are completed
	// 5. Proceed with the change if all approvals are done

	return &v1pb.ApproveResponse{
		ApprovalId: req.ApprovalId,
		Status:     storepb.ApprovalStatus_APPROVAL_STATUS_APPROVED,
	}, nil
}

// Reject implements v1.ApprovalFlowServiceServer.Reject.
func (s *ApprovalFlowService) Reject(ctx context.Context, req *v1pb.RejectRequest) (*v1pb.RejectResponse, error) {
	if req.ApprovalId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "approval_id is required")
	}

	// TODO: Implement rejection logic
	// 1. Get the approval history record
	// 2. Verify the current user has permission to reject
	// 3. Update the approval status to REJECTED
	// 4. Cancel the change

	return &v1pb.RejectResponse{
		ApprovalId: req.ApprovalId,
		Status:     storepb.ApprovalStatus_APPROVAL_STATUS_REJECTED,
	}, nil
}

// ListApprovalHistories implements v1.ApprovalFlowServiceServer.ListApprovalHistories.
func (s *ApprovalFlowService) ListApprovalHistories(ctx context.Context, req *v1pb.ListApprovalHistoriesRequest) (*v1pb.ListApprovalHistoriesResponse, error) {
	find := &store.FindApprovalHistoryMessage{}
	if req.IssueId != 0 {
		find.IssueID = &req.IssueId
	}
	if req.ApprovalId != "" {
		find.ApprovalID = &req.ApprovalId
	}
	if req.Status != storepb.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED {
		find.Status = &req.Status
	}

	histories, err := s.store.ListApprovalHistories(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list approval histories: %v", err)
	}

	return &v1pb.ListApprovalHistoriesResponse{
		Histories:    histories,
		NextPageToken: "", // TODO: Implement pagination
		TotalSize:     int32(len(histories)),
	}, nil
}
