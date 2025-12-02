package v1

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SensitiveApprovalService implements the SensitiveApprovalService API.
type SensitiveApprovalService struct {
	v1.UnimplementedSensitiveApprovalServiceServer
	store *store.Store
}

// NewSensitiveApprovalService creates a new SensitiveApprovalService.
func NewSensitiveApprovalService(store *store.Store) *SensitiveApprovalService {
	return &SensitiveApprovalService{
		store: store,
	}
}

// ListSensitiveLevels lists all sensitive levels.
func (s *SensitiveApprovalService) ListSensitiveLevels(ctx context.Context, req *v1.ListSensitiveLevelsRequest) (*v1.ListSensitiveLevelsResponse, error) {
	// Get the sensitive levels from the store
	sensitiveLevels, err := s.store.ListSensitiveLevels(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sensitive levels: %v", err)
	}

	return &v1.ListSensitiveLevelsResponse{
		SensitiveLevels: sensitiveLevels,
	}, nil
}

// GetSensitiveLevel gets a sensitive level by name.
func (s *SensitiveApprovalService) GetSensitiveLevel(ctx context.Context, req *v1.GetSensitiveLevelRequest) (*v1.SensitiveLevel, error) {
	// Extract sensitive level ID from name
	parts := strings.Split(req.Name, "/")
	if len(parts) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sensitive level name: %s", req.Name)
	}
	sensitiveLevelID := parts[1]

	// Get the sensitive level from the store
	sensitiveLevel, err := s.store.GetSensitiveLevel(ctx, sensitiveLevelID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "sensitive level not found: %s", sensitiveLevelID)
		}
		return nil, status.Errorf(codes.Internal, "failed to get sensitive level: %v", err)
	}

	return sensitiveLevel, nil
}

// CreateSensitiveLevel creates a new sensitive level.
func (s *SensitiveApprovalService) CreateSensitiveLevel(ctx context.Context, req *v1.CreateSensitiveLevelRequest) (*v1.SensitiveLevel, error) {
	// Validate required fields
	if req.SensitiveLevel == nil {
		return nil, status.Errorf(codes.InvalidArgument, "sensitive level is required")
	}
	if req.SensitiveLevel.DisplayName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "sensitive level display name is required")
	}
	if req.SensitiveLevel.Severity == v1.SensitiveLevel_SEVERITY_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "sensitive level severity is required")
	}

	// Generate unique ID
	sensitiveLevelID := uuid.New().String()
	now := timestamppb.Now()

	// Create the sensitive level
	sensitiveLevel := &v1.SensitiveLevel{
		Name:           fmt.Sprintf("sensitive-levels/%s", sensitiveLevelID),
		DisplayName:    req.SensitiveLevel.DisplayName,
		Severity:       req.SensitiveLevel.Severity,
		Description:    req.SensitiveLevel.Description,
		Color:          req.SensitiveLevel.Color,
		FieldMatchRules: req.SensitiveLevel.FieldMatchRules,
		CreateTime:     now,
		UpdateTime:     now,
	}

	// Save to store
	err := s.store.CreateSensitiveLevel(ctx, sensitiveLevel)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create sensitive level: %v", err)
	}

	return sensitiveLevel, nil
}

// UpdateSensitiveLevel updates an existing sensitive level.
func (s *SensitiveApprovalService) UpdateSensitiveLevel(ctx context.Context, req *v1.UpdateSensitiveLevelRequest) (*v1.SensitiveLevel, error) {
	// Extract sensitive level ID from name
	parts := strings.Split(req.SensitiveLevel.Name, "/")
	if len(parts) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sensitive level name: %s", req.SensitiveLevel.Name)
	}
	sensitiveLevelID := parts[1]

	// Get existing sensitive level
	existing, err := s.store.GetSensitiveLevel(ctx, sensitiveLevelID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "sensitive level not found: %s", sensitiveLevelID)
		}
		return nil, status.Errorf(codes.Internal, "failed to get sensitive level: %v", err)
	}

	// Apply updates using field mask
	if req.UpdateMask != nil {
		for _, field := range req.UpdateMask.Paths {
			switch field {
			case "display_name":
				existing.DisplayName = req.SensitiveLevel.DisplayName
			case "severity":
				existing.Severity = req.SensitiveLevel.Severity
			case "description":
				existing.Description = req.SensitiveLevel.Description
			case "color":
				existing.Color = req.SensitiveLevel.Color
			case "field_match_rules":
				existing.FieldMatchRules = req.SensitiveLevel.FieldMatchRules
			}
		}
	}

	// Update timestamp
	existing.UpdateTime = timestamppb.Now()

	// Save to store
	err = s.store.UpdateSensitiveLevel(ctx, existing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update sensitive level: %v", err)
	}

	return existing, nil
}

// DeleteSensitiveLevel deletes a sensitive level.
func (s *SensitiveApprovalService) DeleteSensitiveLevel(ctx context.Context, req *v1.DeleteSensitiveLevelRequest) (*v1.Empty, error) {
	// Extract sensitive level ID from name
	parts := strings.Split(req.Name, "/")
	if len(parts) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sensitive level name: %s", req.Name)
	}
	sensitiveLevelID := parts[1]

	// Delete from store
	err := s.store.DeleteSensitiveLevel(ctx, sensitiveLevelID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "sensitive level not found: %s", sensitiveLevelID)
		}
		return nil, status.Errorf(codes.Internal, "failed to delete sensitive level: %v", err)
	}

	return &v1.Empty{}, nil
}

// ListApprovalFlows lists all approval flows.
func (s *SensitiveApprovalService) ListApprovalFlows(ctx context.Context, req *v1.ListApprovalFlowsRequest) (*v1.ListApprovalFlowsResponse, error) {
	// Get the approval flows from the store
	approvalFlows, err := s.store.ListApprovalFlows(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list approval flows: %v", err)
	}

	return &v1.ListApprovalFlowsResponse{
		ApprovalFlows: approvalFlows,
	}, nil
}

// GetApprovalFlow gets an approval flow by name.
func (s *SensitiveApprovalService) GetApprovalFlow(ctx context.Context, req *v1.GetApprovalFlowRequest) (*v1.ApprovalFlow, error) {
	// Extract approval flow ID from name
	parts := strings.Split(req.Name, "/")
	if len(parts) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid approval flow name: %s", req.Name)
	}
	approvalFlowID := parts[1]

	// Get the approval flow from the store
	approvalFlow, err := s.store.GetApprovalFlow(ctx, approvalFlowID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "approval flow not found: %s", approvalFlowID)
		}
		return nil, status.Errorf(codes.Internal, "failed to get approval flow: %v", err)
	}

	return approvalFlow, nil
}

// CreateApprovalFlow creates a new approval flow.
func (s *SensitiveApprovalService) CreateApprovalFlow(ctx context.Context, req *v1.CreateApprovalFlowRequest) (*v1.ApprovalFlow, error) {
	// Validate required fields
	if req.ApprovalFlow == nil {
		return nil, status.Errorf(codes.InvalidArgument, "approval flow is required")
	}
	if req.ApprovalFlow.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "approval flow name is required")
	}
	if req.ApprovalFlow.SensitiveSeverity == v1.SensitiveLevel_SEVERITY_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "approval flow sensitive severity is required")
	}
	if len(req.ApprovalFlow.Steps) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "approval flow must have at least one step")
	}

	// Generate unique ID
	approvalFlowID := uuid.New().String()
	now := timestamppb.Now()

	// Create the approval flow
	approvalFlow := &v1.ApprovalFlow{
		Name:            fmt.Sprintf("approval-flows/%s", approvalFlowID),
		DisplayName:     req.ApprovalFlow.DisplayName,
		Description:     req.ApprovalFlow.Description,
		SensitiveSeverity: req.ApprovalFlow.SensitiveSeverity,
		Steps:           req.ApprovalFlow.Steps,
		CreateTime:      now,
		UpdateTime:      now,
	}

	// Save to store
	err := s.store.CreateApprovalFlow(ctx, approvalFlow)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create approval flow: %v", err)
	}

	return approvalFlow, nil
}

// UpdateApprovalFlow updates an existing approval flow.
func (s *SensitiveApprovalService) UpdateApprovalFlow(ctx context.Context, req *v1.UpdateApprovalFlowRequest) (*v1.ApprovalFlow, error) {
	// Extract approval flow ID from name
	parts := strings.Split(req.ApprovalFlow.Name, "/")
	if len(parts) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid approval flow name: %s", req.ApprovalFlow.Name)
	}
	approvalFlowID := parts[1]

	// Get existing approval flow
	existing, err := s.store.GetApprovalFlow(ctx, approvalFlowID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "approval flow not found: %s", approvalFlowID)
		}
		return nil, status.Errorf(codes.Internal, "failed to get approval flow: %v", err)
	}

	// Apply updates using field mask
	if req.UpdateMask != nil {
		for _, field := range req.UpdateMask.Paths {
			switch field {
			case "display_name":
				existing.DisplayName = req.ApprovalFlow.DisplayName
			case "description":
				existing.Description = req.ApprovalFlow.Description
			case "sensitive_severity":
				existing.SensitiveSeverity = req.ApprovalFlow.SensitiveSeverity
			case "steps":
				existing.Steps = req.ApprovalFlow.Steps
			}
		}
	}

	// Update timestamp
	existing.UpdateTime = timestamppb.Now()

	// Save to store
	err = s.store.UpdateApprovalFlow(ctx, existing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update approval flow: %v", err)
	}

	return existing, nil
}

// DeleteApprovalFlow deletes an approval flow.
func (s *SensitiveApprovalService) DeleteApprovalFlow(ctx context.Context, req *v1.DeleteApprovalFlowRequest) (*v1.Empty, error) {
	// Extract approval flow ID from name
	parts := strings.Split(req.Name, "/")
	if len(parts) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid approval flow name: %s", req.Name)
	}
	approvalFlowID := parts[1]

	// Delete from store
	err := s.store.DeleteApprovalFlow(ctx, approvalFlowID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "approval flow not found: %s", approvalFlowID)
		}
		return nil, status.Errorf(codes.Internal, "failed to delete approval flow: %v", err)
	}

	return &v1.Empty{}, nil
}
