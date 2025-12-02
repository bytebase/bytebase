package interceptor

import (
	"context"
	"fmt"
	"log"

	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/service"
)

// SensitiveDataInterceptor is the interceptor for sensitive data changes.
// It checks if a change involves sensitive data and requires approval.
// If approval is required, it creates an approval request and blocks the change until approved.
type SensitiveDataInterceptor interface {
	// InterceptChange intercepts a database change and checks if it requires approval.
	// If approval is required, it returns an approval request and an error indicating that approval is needed.
	// If approval is not required, it returns nil, nil.
	// If there's an error processing the request, it returns nil and the error.
	InterceptChange(ctx context.Context, change *service.DatabaseChange) (*v1.ApprovalRequest, error)

	// CanExecuteChange checks if a database change can be executed (i.e., it has been approved or doesn't require approval).
	// If the change can be executed, it returns true, nil.
	// If the change cannot be executed (e.g., it requires approval but hasn't been approved), it returns false, nil.
	// If there's an error processing the request, it returns false and the error.
	CanExecuteChange(ctx context.Context, changeID string) (bool, error)
}

// sensitiveDataInterceptorImpl is the implementation of SensitiveDataInterceptor.
type sensitiveDataInterceptorImpl struct {
	sensitiveDataService  service.SensitiveDataService
	approvalService       service.ApprovalService
	auditLogService       service.AuditLogService
	notificationService   service.NotificationService
}

// NewSensitiveDataInterceptor creates a new SensitiveDataInterceptor.
func NewSensitiveDataInterceptor(
	sensitiveDataService service.SensitiveDataService,
	approvalService service.ApprovalService,
	auditLogService service.AuditLogService,
	notificationService service.NotificationService,
) SensitiveDataInterceptor {
	return &sensitiveDataInterceptorImpl{
		sensitiveDataService:  sensitiveDataService,
		approvalService:       approvalService,
		auditLogService:       auditLogService,
		notificationService:   notificationService,
	}
}

// InterceptChange intercepts a database change and checks if it requires approval.
func (s *sensitiveDataInterceptorImpl) InterceptChange(ctx context.Context, change *service.DatabaseChange) (*v1.ApprovalRequest, error) {
	log.Infof("Intercepting database change: %s", change.ChangeID)

	// Detect sensitive data in the change
	matches, err := s.sensitiveDataService.DetectSensitiveData(ctx, change)
	if err != nil {
		log.Errorf("Failed to detect sensitive data: %v", err)
		return nil, err
	}

	// Get maximum sensitive severity
	maxSeverity := s.sensitiveDataService.GetMaxSensitiveSeverity(matches)

	// Check if the change requires approval
	requiresApproval := service.NeedsApproval(maxSeverity)
	if !requiresApproval {
		log.Infof("Change %s does not require approval (sensitivity: %v)", change.ChangeID, changeSensitivity.HighestSeverity)
		// Log the change in audit logs
		s.auditLogService.Log(ctx, &service.AuditLogEntry{
			Action:            service.AuditLogAction_CHANGE_DETECTED,
			ChangeID:          change.ChangeID,
			Description:       fmt.Sprintf("Change detected, no approval required (sensitivity: %v)", changeSensitivity.HighestSeverity),
			Actor:             change.Requester,
			ResourceType:      service.AuditLogResourceType_DATABASE_CHANGE,
			ResourceID:        change.ChangeID,
			SensitiveSeverity: changeSensitivity.HighestSeverity,
		})
		return nil, nil
	}

	// Create approval request
	approvalRequest, err := s.approvalService.CreateApprovalRequest(ctx, &service.CreateApprovalRequest{
		DisplayName:       fmt.Sprintf("Change %s (sensitivity: %v)", change.ChangeID, changeSensitivity.HighestSeverity),
		Description:       change.Description,
		Requester:         change.Requester,
		SensitiveSeverity: changeSensitivity.HighestSeverity,
		Details: &v1.ApprovalRequestDetails{
			ChangeID:      change.ChangeID,
			Database:      change.Database,
			Table:         change.Table,
			Schema:        change.Schema,
			SQL:           change.SQL,
			SensitiveData: changeSensitivity.SensitiveData,
		},
		// ApprovalFlowID is empty, approval service will find appropriate flow based on sensitivity
	})
	if err != nil {
		log.Errorf("Failed to create approval request for change %s: %v", change.ChangeID, err)
		return nil, err
	}

	// Log the change and approval request in audit logs
	s.auditLogService.Log(ctx, &service.AuditLogEntry{
		Action:            service.AuditLogAction_CHANGE_DETECTED,
		ChangeID:          change.ChangeID,
		ApprovalRequestID: approvalRequest.Name,
		Description:       fmt.Sprintf("Change detected, requires approval (sensitivity: %v)", changeSensitivity.HighestSeverity),
		Actor:             change.Requester,
		ResourceType:      service.AuditLogResourceType_DATABASE_CHANGE,
		ResourceID:        change.ChangeID,
		SensitiveSeverity: changeSensitivity.HighestSeverity,
	})

	// Notify requester that approval is required
	s.notificationService.Notify(ctx, change.Requester, "Change Requires Approval", fmt.Sprintf("Your change '%s' involves sensitive data and requires approval. Please wait for approval before execution.", change.ChangeID))

	log.Infof("Change %s requires approval. Approval request created: %s", change.ChangeID, approvalRequest.Name)
	return approvalRequest, fmt.Errorf("change involves sensitive data and requires approval. approval request: %s", approvalRequest.Name)
}

// CanExecuteChange checks if a database change can be executed.
func (s *sensitiveDataInterceptorImpl) CanExecuteChange(ctx context.Context, changeID string) (bool, error) {
	log.Infof("Checking if change %s can be executed", changeID)

	// Get approval request for the change
	// Note: In a real implementation, you would need to store the mapping between changeID and approvalRequestID
	// For now, we'll assume we can find the approval request by changeID
	// TODO: Implement a way to find approval request by changeID

	// For this example, we'll assume the change can be executed if no approval is required or if the approval request is approved
	// In a real implementation, you would need to:
	// 1. Check if the change requires approval
	// 2. If it does, find the corresponding approval request
	// 3. Check if the approval request is in approved status
	// 4. If it's not approved, return false

	log.Infof("Change %s can be executed (mock response)", changeID)
	return true, nil
}
