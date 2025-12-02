// Copyright 2024 Bytebase Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// ApprovalFlowService implements the approval flow service.
type ApprovalFlowService struct {
	v1connect.UnimplementedApprovalFlowServiceHandler
	store          *store.Store
	stateCfg       *state.State
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewApprovalFlowService creates a new ApprovalFlowService.
func NewApprovalFlowService(
	store *store.Store,
	stateCfg *state.State,
	licenseService *enterprise.LicenseService,
	profile *config.Profile,
	iamManager *iam.Manager,
) *ApprovalFlowService {
	return &ApprovalFlowService{
		store:          store,
		stateCfg:       stateCfg,
		licenseService: licenseService,
		profile:        profile,
	iamManager:     iamManager,
	}
}

// CreateApprovalFlow creates a new approval flow configuration.
func (s *ApprovalFlowService) CreateApprovalFlow(ctx context.Context, req *connect.Request[v1pb.CreateApprovalFlowRequest]) (*connect.Response[v1pb.ApprovalFlow], error) {
	// Validate request
	if req.Msg.ApprovalFlow == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("approval_flow is required"))
	}

	if req.Msg.ApprovalFlow.DisplayName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("display_name is required"))
	}

	if req.Msg.ApprovalFlow.SensitivityLevel == v1pb.SensitivityLevel_SENSITIVITY_LEVEL_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sensitivity_level is required"))
	}

	if len(req.Msg.ApprovalFlow.Steps) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("steps is required"))
	}

	// Create approval flow
	approvalFlow := &store.ApprovalFlowMessage{
		DisplayName:      req.Msg.ApprovalFlow.DisplayName,
		Description:      req.Msg.ApprovalFlow.Description,
		SensitivityLevel: int32(req.Msg.ApprovalFlow.SensitivityLevel),
		Steps:            convertApprovalStepsToStore(req.Msg.ApprovalFlow.Steps),
	}

	createdApprovalFlow, err := s.store.CreateApprovalFlow(ctx, approvalFlow)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create approval flow: %v", err))
	}

	// Convert to v1pb.ApprovalFlow
	result := convertToApprovalFlow(ctx, createdApprovalFlow)

	return connect.NewResponse(result), nil
}

// GetApprovalFlow gets an approval flow configuration.
func (s *ApprovalFlowService) GetApprovalFlow(ctx context.Context, req *connect.Request[v1pb.GetApprovalFlowRequest]) (*connect.Response[v1pb.ApprovalFlow], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract approval flow ID from name
	parts := strings.Split(req.Msg.Name, "/")
	if len(parts) != 2 || parts[0] != "approvalFlows" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.Name))
	}

	id := parts[1]

	// Get approval flow
	approvalFlow, err := s.store.GetApprovalFlow(ctx, &store.FindApprovalFlowMessage{ID: &id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get approval flow: %v", err))
	}

	if approvalFlow == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("approval flow %s not found", id))
	}

	// Convert to v1pb.ApprovalFlow
	result := convertToApprovalFlow(ctx, approvalFlow)

	return connect.NewResponse(result), nil
}

// ListApprovalFlows lists all approval flow configurations.
func (s *ApprovalFlowService) ListApprovalFlows(ctx context.Context, req *connect.Request[v1pb.ListApprovalFlowsRequest]) (*connect.Response[v1pb.ListApprovalFlowsResponse], error) {
	// Build find message
	find := &store.FindApprovalFlowMessage{}

	if req.Msg.SensitivityLevel != v1pb.SensitivityLevel_SENSITIVITY_LEVEL_UNSPECIFIED {
		level := int32(req.Msg.SensitivityLevel)
		find.SensitivityLevel = &level
	}

	// Set pagination
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 100
	}
	find.Limit = &limit

	// Get approval flows
	approvalFlows, err := s.store.ListApprovalFlows(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list approval flows: %v", err))
	}

	// Convert to v1pb.ApprovalFlow
	result := &v1pb.ListApprovalFlowsResponse{}
	for _, af := range approvalFlows {
		result.ApprovalFlows = append(result.ApprovalFlows, convertToApprovalFlow(ctx, af))
	}

	// Set next page token if needed
	if len(approvalFlows) == limit {
		result.NextPageToken = fmt.Sprintf("%d", limit)
	}

	return connect.NewResponse(result), nil
}

// UpdateApprovalFlow updates an approval flow configuration.
func (s *ApprovalFlowService) UpdateApprovalFlow(ctx context.Context, req *connect.Request[v1pb.UpdateApprovalFlowRequest]) (*connect.Response[v1pb.ApprovalFlow], error) {
	if req.Msg.ApprovalFlow == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("approval_flow is required"))
	}

	if req.Msg.ApprovalFlow.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract approval flow ID from name
	parts := strings.Split(req.Msg.ApprovalFlow.Name, "/")
	if len(parts) != 2 || parts[0] != "approvalFlows" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.ApprovalFlow.Name))
	}

	id := parts[1]

	// Get existing approval flow
	existing, err := s.store.GetApprovalFlow(ctx, &store.FindApprovalFlowMessage{ID: &id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get approval flow: %v", err))
	}

	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("approval flow %s not found", id))
	}

	// Build update message
	update := &store.UpdateApprovalFlowMessage{}

	// Apply field mask
	for _, field := range req.Msg.UpdateMask.Paths {
		switch field {
		case "display_name":
			update.DisplayName = &req.Msg.ApprovalFlow.DisplayName
		case "description":
			update.Description = &req.Msg.ApprovalFlow.Description
		case "sensitivity_level":
			level := int32(req.Msg.ApprovalFlow.SensitivityLevel)
			update.SensitivityLevel = &level
		case "steps":
			update.Steps = convertApprovalStepsToStore(req.Msg.ApprovalFlow.Steps)
		}
	}

	// Update approval flow
	updatedApprovalFlow, err := s.store.UpdateApprovalFlow(ctx, id, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update approval flow: %v", err))
	}

	// Convert to v1pb.ApprovalFlow
	result := convertToApprovalFlow(ctx, updatedApprovalFlow)

	return connect.NewResponse(result), nil
}

// DeleteApprovalFlow deletes an approval flow configuration.
func (s *ApprovalFlowService) DeleteApprovalFlow(ctx context.Context, req *connect.Request[v1pb.DeleteApprovalFlowRequest]) (*connect.Response[v1pb.Empty], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract approval flow ID from name
	parts := strings.Split(req.Msg.Name, "/")
	if len(parts) != 2 || parts[0] != "approvalFlows" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.Name))
	}

	id := parts[1]

	// Delete approval flow
	err := s.store.DeleteApprovalFlow(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to delete approval flow: %v", err))
	}

	return connect.NewResponse(&v1pb.Empty{}), nil
}

// SubmitApproval submits a change for approval.
func (s *ApprovalFlowService) SubmitApproval(ctx context.Context, req *connect.Request[v1pb.SubmitApprovalRequest]) (*connect.Response[v1pb.ApprovalRequest], error) {
	// Validate request
	if req.Msg.ApprovalRequest == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("approval_request is required"))
	}

	if req.Msg.ApprovalRequest.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("title is required"))
	}

	if req.Msg.ApprovalRequest.IssueId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("issue_id is required"))
	}

	if req.Msg.ApprovalRequest.SensitivityLevel == v1pb.SensitivityLevel_SENSITIVITY_LEVEL_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sensitivity_level is required"))
	}

	if req.Msg.ApprovalRequest.ApprovalFlowName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("approval_flow_name is required"))
	}

	// Get approval flow
	approvalFlow, err := s.store.GetApprovalFlow(ctx, &store.FindApprovalFlowMessage{ID: &req.Msg.ApprovalRequest.ApprovalFlowName})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get approval flow: %v", err))
	}

	if approvalFlow == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("approval flow %s not found", req.Msg.ApprovalRequest.ApprovalFlowName))
	}

	// Create approval request
	approvalRequest := &store.ApprovalRequestMessage{
		Title:            req.Msg.ApprovalRequest.Title,
		Description:      req.Msg.ApprovalRequest.Description,
		IssueID:          req.Msg.ApprovalRequest.IssueId,
		SensitivityLevel: int32(req.Msg.ApprovalRequest.SensitivityLevel),
		ApprovalFlowID:   req.Msg.ApprovalRequest.ApprovalFlowName,
		Status:           int32(v1pb.ApprovalStatus_APPROVAL_STATUS_PENDING),
		Submitter:        req.Msg.ApprovalRequest.Submitter,
	}

	createdApprovalRequest, err := s.store.CreateApprovalRequest(ctx, approvalRequest)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create approval request: %v", err))
	}

	// Convert to v1pb.ApprovalRequest
	result := convertToApprovalRequest(ctx, createdApprovalRequest)

	return connect.NewResponse(result), nil
}

// GetApprovalRequest gets an approval request.
func (s *ApprovalFlowService) GetApprovalRequest(ctx context.Context, req *connect.Request[v1pb.GetApprovalRequest]) (*connect.Response[v1pb.ApprovalRequest], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract approval request ID from name
	parts := strings.Split(req.Msg.Name, "/")
	if len(parts) != 2 || parts[0] != "approvalRequests" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.Name))
	}

	id := parts[1]

	// Get approval request
	approvalRequest, err := s.store.GetApprovalRequest(ctx, &store.FindApprovalRequestMessage{ID: &id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get approval request: %v", err))
	}

	if approvalRequest == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("approval request %s not found", id))
	}

	// Convert to v1pb.ApprovalRequest
	result := convertToApprovalRequest(ctx, approvalRequest)

	return connect.NewResponse(result), nil
}

// ListApprovalRequests lists all approval requests.
func (s *ApprovalFlowService) ListApprovalRequests(ctx context.Context, req *connect.Request[v1pb.ListApprovalRequestsRequest]) (*connect.Response[v1pb.ListApprovalRequestsResponse], error) {
	// Build find message
	find := &store.FindApprovalRequestMessage{}

	if req.Msg.IssueId != "" {
		find.IssueID = &req.Msg.IssueId
	}

	if req.Msg.SensitivityLevel != v1pb.SensitivityLevel_SENSITIVITY_LEVEL_UNSPECIFIED {
		level := int32(req.Msg.SensitivityLevel)
		find.SensitivityLevel = &level
	}

	if req.Msg.Status != v1pb.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED {
		status := int32(req.Msg.Status)
		find.Status = &status
	}

	if req.Msg.Submitter != "" {
		find.Submitter = &req.Msg.Submitter
	}

	// Set pagination
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 100
	}
	find.Limit = &limit

	// Get approval requests
	approvalRequests, err := s.store.ListApprovalRequests(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list approval requests: %v", err))
	}

	// Convert to v1pb.ApprovalRequest
	result := &v1pb.ListApprovalRequestsResponse{}
	for _, ar := range approvalRequests {
		result.ApprovalRequests = append(result.ApprovalRequests, convertToApprovalRequest(ctx, ar))
	}

	// Set next page token if needed
	if len(approvalRequests) == limit {
		result.NextPageToken = fmt.Sprintf("%d", limit)
	}

	return connect.NewResponse(result), nil
}

// ApproveRequest approves an approval request.
func (s *ApprovalFlowService) ApproveRequest(ctx context.Context, req *connect.Request[v1pb.ApproveRequest]) (*connect.Response[v1pb.ApprovalRequest], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract approval request ID from name
	parts := strings.Split(req.Msg.Name, "/")
	if len(parts) != 2 || parts[0] != "approvalRequests" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.Name))
	}

	id := parts[1]

	// Get approval request
	approvalRequest, err := s.store.GetApprovalRequest(ctx, &store.FindApprovalRequestMessage{ID: &id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get approval request: %v", err))
	}

	if approvalRequest == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("approval request %s not found", id))
	}

	// Update status to approved
	update := &store.UpdateApprovalRequestMessage{
		Status: int32(v1pb.ApprovalStatus_APPROVAL_STATUS_APPROVED),
	}

	updatedApprovalRequest, err := s.store.UpdateApprovalRequest(ctx, id, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to approve request: %v", err))
	}

	// Convert to v1pb.ApprovalRequest
	result := convertToApprovalRequest(ctx, updatedApprovalRequest)

	return connect.NewResponse(result), nil
}

// RejectRequest rejects an approval request.
func (s *ApprovalFlowService) RejectRequest(ctx context.Context, req *connect.Request[v1pb.RejectRequest]) (*connect.Response[v1pb.ApprovalRequest], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract approval request ID from name
	parts := strings.Split(req.Msg.Name, "/")
	if len(parts) != 2 || parts[0] != "approvalRequests" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.Name))
	}

	id := parts[1]

	// Get approval request
	approvalRequest, err := s.store.GetApprovalRequest(ctx, &store.FindApprovalRequestMessage{ID: &id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get approval request: %v", err))
	}

	if approvalRequest == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("approval request %s not found", id))
	}

	// Update status to rejected
	update := &store.UpdateApprovalRequestMessage{
		Status: int32(v1pb.ApprovalStatus_APPROVAL_STATUS_REJECTED),
	}

	updatedApprovalRequest, err := s.store.UpdateApprovalRequest(ctx, id, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to reject request: %v", err))
	}

	// Convert to v1pb.ApprovalRequest
	result := convertToApprovalRequest(ctx, updatedApprovalRequest)

	return connect.NewResponse(result), nil
}

// convertToApprovalFlow converts a store.ApprovalFlowMessage to v1pb.ApprovalFlow.
func convertToApprovalFlow(ctx context.Context, af *store.ApprovalFlowMessage) *v1pb.ApprovalFlow {
	if af == nil {
		return nil
	}

	return &v1pb.ApprovalFlow{
		Name:              fmt.Sprintf("approvalFlows/%s", af.ID),
		DisplayName:       af.DisplayName,
		Description:       af.Description,
		SensitivityLevel:  v1pb.SensitivityLevel(af.SensitivityLevel),
		Steps:             convertApprovalStepsFromStore(af.Steps),
		CreateTime:        utils.TimestampFromTime(af.CreatedAt),
		UpdateTime:        utils.TimestampFromTime(af.UpdatedAt),
	}
}

// convertToApprovalRequest converts a store.ApprovalRequestMessage to v1pb.ApprovalRequest.
func convertToApprovalRequest(ctx context.Context, ar *store.ApprovalRequestMessage) *v1pb.ApprovalRequest {
	if ar == nil {
		return nil
	}

	return &v1pb.ApprovalRequest{
		Name:              fmt.Sprintf("approvalRequests/%s", ar.ID),
		Title:             ar.Title,
		Description:       ar.Description,
		IssueId:           ar.IssueID,
		SensitivityLevel:  v1pb.SensitivityLevel(ar.SensitivityLevel),
		ApprovalFlowName:  ar.ApprovalFlowID,
		Status:            v1pb.ApprovalStatus(ar.Status),
		Submitter:         ar.Submitter,
		CreateTime:        utils.TimestampFromTime(ar.CreatedAt),
		UpdateTime:        utils.TimestampFromTime(ar.UpdatedAt),
	}
}

// convertApprovalStepsFromStore converts store.ApprovalStepMessage to v1pb.ApprovalStep.
func convertApprovalStepsFromStore(steps []*store.ApprovalStepMessage) []*v1pb.ApprovalStep {
	result := make([]*v1pb.ApprovalStep, 0, len(steps))
	for _, step := range steps {
		result = append(result, &v1pb.ApprovalStep{
			Name:           step.Name,
			Description:    step.Description,
			Role:           step.Role,
			Order:          step.Order,
			MinApprovals:   step.MinApprovals,
			MaxApprovals:   step.MaxApprovals,
		})
	}
	return result
}

// convertApprovalStepsToStore converts v1pb.ApprovalStep to store.ApprovalStepMessage.
func convertApprovalStepsToStore(steps []*v1pb.ApprovalStep) []*store.ApprovalStepMessage {
	result := make([]*store.ApprovalStepMessage, 0, len(steps))
	for _, step := range steps {
		result = append(result, &store.ApprovalStepMessage{
			Name:           step.Name,
			Description:    step.Description,
			Role:           step.Role,
			Order:          step.Order,
			MinApprovals:   step.MinApprovals,
			MaxApprovals:   step.MaxApprovals,
		})
	}
	return result
}