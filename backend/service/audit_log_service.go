package service

import (
	"context"
	"log"

	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// AuditLogAction is the type of audit log action.
type AuditLogAction string

// Audit log actions
const (
	AuditLogAction_CHANGE_DETECTED       AuditLogAction = "CHANGE_DETECTED"
	AuditLogAction_CHANGE_APPROVED       AuditLogAction = "CHANGE_APPROVED"
	AuditLogAction_CHANGE_REJECTED       AuditLogAction = "CHANGE_REJECTED"
	AuditLogAction_CHANGE_EXECUTED       AuditLogAction = "CHANGE_EXECUTED"
	AuditLogAction_APPROVAL_REQUEST_CREATED AuditLogAction = "APPROVAL_REQUEST_CREATED"
	AuditLogAction_APPROVAL_REQUEST_APPROVED AuditLogAction = "APPROVAL_REQUEST_APPROVED"
	AuditLogAction_APPROVAL_REQUEST_REJECTED AuditLogAction = "APPROVAL_REQUEST_REJECTED"
)

// AuditLogResourceType is the type of resource associated with an audit log entry.
type AuditLogResourceType string

// Audit log resource types
const (
	AuditLogResourceType_DATABASE_CHANGE AuditLogResourceType = "DATABASE_CHANGE"
	AuditLogResourceType_APPROVAL_REQUEST AuditLogResourceType = "APPROVAL_REQUEST"
	AuditLogResourceType_SENSITIVE_LEVEL AuditLogResourceType = "SENSITIVE_LEVEL"
	AuditLogResourceType_APPROVAL_FLOW   AuditLogResourceType = "APPROVAL_FLOW"
)

// AuditLogEntry is an entry in the audit log.
type AuditLogEntry struct {
	Action            AuditLogAction
	ChangeID          string
	ApprovalRequestID string
	Description       string
	Actor             string
	ResourceType      AuditLogResourceType
	ResourceID        string
	SensitiveSeverity v1.SensitiveLevel_Severity
	Metadata          map[string]interface{}
}

// AuditLogService is the service for managing audit logs.
type AuditLogService interface {
	// Log creates a new audit log entry.
	Log(ctx context.Context, entry *AuditLogEntry) error

	// ListAuditLogs lists audit log entries.
	ListAuditLogs(ctx context.Context, filters *ListAuditLogsFilters) ([]*AuditLogEntry, error)
}

// ListAuditLogsFilters is the filter for listing audit logs.
type ListAuditLogsFilters struct {
	Action            *AuditLogAction
	ChangeID          *string
	ApprovalRequestID *string
	Actor             *string
	ResourceType      *AuditLogResourceType
	ResourceID        *string
	SensitiveSeverity *v1.SensitiveLevel_Severity
	StartTime         *int64
	EndTime           *int64
}

// auditLogServiceImpl is the implementation of AuditLogService.
type auditLogServiceImpl struct {
	// In a real implementation, you would have a store to persist audit logs
}

// NewAuditLogService creates a new AuditLogService.
func NewAuditLogService() AuditLogService {
	return &auditLogServiceImpl{}
}

// Log creates a new audit log entry.
func (s *auditLogServiceImpl) Log(ctx context.Context, entry *AuditLogEntry) error {
	log.Infof("Audit log - %s: %s (Actor: %s, Resource: %s/%s, Sensitivity: %v)",
		entry.Action,
		entry.Description,
		entry.Actor,
		entry.ResourceType,
		entry.ResourceID,
		entry.SensitiveSeverity,
	)

	// In a real implementation, you would persist the audit log entry to a database
	// TODO: Implement persistence of audit log entries

	return nil
}

// ListAuditLogs lists audit log entries.
func (s *auditLogServiceImpl) ListAuditLogs(ctx context.Context, filters *ListAuditLogsFilters) ([]*AuditLogEntry, error) {
	log.Infof("Listing audit logs with filters: %+v", filters)

	// In a real implementation, you would query the database for audit log entries based on the filters
	// TODO: Implement listing of audit log entries

	return nil, nil
}
