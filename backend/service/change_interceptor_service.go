package service

import (
	"context"
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/common/log"
)

// ChangeInterceptorService is the service for intercepting database changes and enforcing approval policies.
type ChangeInterceptorService interface {
	// InterceptChange checks if a database change requires approval and intercepts it if necessary
	InterceptChange(ctx context.Context, req *ChangeInterceptRequest) (*ChangeInterceptResponse, error)
	// CheckApprovalRequired checks if a change requires approval
	CheckApprovalRequired(ctx context.Context, req *ChangeInterceptRequest) (bool, []*SensitiveDataMatch, error)
}

// ChangeInterceptRequest is the request for intercepting a database change.
type ChangeInterceptRequest struct {
	SQL            string
	DatabaseInstance string
	Database       string
	Schema         string
	Table          string
	UserID         string
	ProjectID      string
	Environment    string
}

// ChangeInterceptResponse is the response for change interception.
type ChangeInterceptResponse struct {
	RequiresApproval bool
	SensitiveMatches []*SensitiveDataMatch
	ApprovalFlow     *v1.ApprovalFlow
	ApprovalRequest  *v1.ApprovalRequest
	Blocked          bool
	Reason           string
}

// changeInterceptorServiceImpl is the implementation of ChangeInterceptorService.
type changeInterceptorServiceImpl struct {
	sensitiveDataService SensitiveDataService
	sensitiveApprovalService v1.SensitiveApprovalServiceClient
	store *store.Store
}

// NewChangeInterceptorService creates a new ChangeInterceptorService.
func NewChangeInterceptorService(
	sensitiveDataService SensitiveDataService,
	sensitiveApprovalService v1.SensitiveApprovalServiceClient,
	store *store.Store,
) ChangeInterceptorService {
	return &changeInterceptorServiceImpl{
		sensitiveDataService: sensitiveDataService,
		sensitiveApprovalService: sensitiveApprovalService,
		store: store,
	}
}

// InterceptChange checks if a database change requires approval and intercepts it if necessary.
func (s *changeInterceptorServiceImpl) InterceptChange(ctx context.Context, req *ChangeInterceptRequest) (*ChangeInterceptResponse, error) {
	log.Infof("Intercepting change: User=%s, Database=%s.%s, Environment=%s", req.UserID, req.DatabaseInstance, req.Database, req.Environment)

	// Check if the change involves sensitive data
	matches, err := s.sensitiveDataService.DetectSensitiveData(ctx, req.SQL, req.DatabaseInstance, req.Database)
	if err != nil {
		log.Errorf("Failed to detect sensitive data: %v", err)
		return nil, err
	}

	if len(matches) == 0 {
		log.Infof("No sensitive data detected, allowing change")
		return &ChangeInterceptResponse{
			RequiresApproval: false,
			Blocked: false,
			Reason: "No sensitive data detected",
		}, nil
	}

	// Get maximum sensitive severity
	maxSeverity := GetMaxSensitiveSeverity(matches)
	log.Infof("Detected sensitive data with maximum severity: %v", maxSeverity)

	// Check if approval is required
	if !NeedsApproval(maxSeverity) {
		log.Infof("Low sensitivity data, no approval required")
		return &ChangeInterceptResponse{
			RequiresApproval: false,
			SensitiveMatches: matches,
			Blocked: false,
			Reason: "Low sensitivity data",
		}, nil
	}

	// Get approval flow for this severity
	approvalFlow, err := s.getApprovalFlowForSeverity(ctx, maxSeverity)
	if err != nil {
		log.Errorf("Failed to get approval flow: %v", err)
		return nil, err
	}

	if approvalFlow == nil {
		log.Infof("No approval flow configured for severity %v, allowing change", maxSeverity)
		return &ChangeInterceptResponse{
			RequiresApproval: false,
			SensitiveMatches: matches,
			Blocked: false,
			Reason: "No approval flow configured",
		}, nil
	}

	// Create approval request
	approvalRequest, err := s.createApprovalRequest(ctx, req, matches, maxSeverity)
	if err != nil {
		log.Errorf("Failed to create approval request: %v", err)
		return nil, err
	}

	// Block the change until approval is granted
	log.Infof("Change blocked, requires approval. Approval Request: %s", approvalRequest.Name)
	return &ChangeInterceptResponse{
		RequiresApproval: true,
		SensitiveMatches: matches,
		ApprovalFlow: approvalFlow,
		ApprovalRequest: approvalRequest,
		Blocked: true,
		Reason: fmt.Sprintf("%s data change requires approval", maxSeverity.String()),
	}, nil
}

// CheckApprovalRequired checks if a change requires approval.
func (s *changeInterceptorServiceImpl) CheckApprovalRequired(ctx context.Context, req *ChangeInterceptRequest) (bool, []*SensitiveDataMatch, error) {
	// Check if the change involves sensitive data
	matches, err := s.sensitiveDataService.DetectSensitiveData(ctx, req.SQL, req.DatabaseInstance, req.Database)
	if err != nil {
		return false, nil, err
	}

	if len(matches) == 0 {
		return false, nil, nil
	}

	// Get maximum sensitive severity
	maxSeverity := GetMaxSensitiveSeverity(matches)

	// Check if approval is required
	return NeedsApproval(maxSeverity), matches, nil
}

// getApprovalFlowForSeverity gets the approval flow for a specific sensitive severity.
func (s *changeInterceptorServiceImpl) getApprovalFlowForSeverity(ctx context.Context, severity v1.SensitiveLevel_Severity) (*v1.ApprovalFlow, error) {
	// List all approval flows
	req := &v1.ListApprovalFlowsRequest{}
	resp, err := s.sensitiveApprovalService.ListApprovalFlows(ctx, req)
	if err != nil {
		return nil, err
	}

	// Find the first approval flow that matches the severity
	for _, flow := range resp.ApprovalFlows {
		if flow.SensitiveSeverity == severity && len(flow.Steps) > 0 {
			return flow, nil
		}
	}

	return nil, nil
}

// createApprovalRequest creates a new approval request.
func (s *changeInterceptorServiceImpl) createApprovalRequest(
	ctx context.Context,
	req *ChangeInterceptRequest,
	matches []*SensitiveDataMatch,
	severity v1.SensitiveLevel_Severity,
) (*v1.ApprovalRequest, error) {
	// Extract sensitive level names
	var sensitiveLevelNames []string
	for _, match := range matches {
		if match.SensitiveLevel != nil {
			sensitiveLevelNames = append(sensitiveLevelNames, match.SensitiveLevel.Name)
		}
	}

	// Create approval request details
	details := &v1.ApprovalRequestDetails{
		SensitiveLevels:    sensitiveLevelNames,
		DatabaseInstance: req.DatabaseInstance,
		Database:         req.Database,
		TableName:        req.Table,
		FieldName:        s.getFieldNamesFromMatches(matches),
		SqlStatement:     req.SQL,
	}

	// Create approval request
	approvalRequest := &v1.ApprovalRequest{
		DisplayName:     fmt.Sprintf("Change Approval: %s.%s", req.DatabaseInstance, req.Database),
		Description:     fmt.Sprintf("SQL change involving %s data", severity.String()),
		Requester:       req.UserID,
		Status:          v1.ApprovalRequest_STATUS_PENDING,
		SensitiveSeverity: severity,
		Details:         details,
	}

	// Save approval request to store
	// TODO: Implement store creation

	log.Infof("Created approval request: %s", approvalRequest.DisplayName)
	return approvalRequest, nil
}

// getFieldNamesFromMatches extracts field names from sensitive data matches.
func (s *changeInterceptorServiceImpl) getFieldNamesFromMatches(matches []*SensitiveDataMatch) []string {
	fieldNames := make(map[string]bool)
	for _, match := range matches {
		fieldNames[match.FieldName] = true
	}

	var result []string
	for field := range fieldNames {
		result = append(result, field)
	}

	return result
}
