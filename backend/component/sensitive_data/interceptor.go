package sensitive_data

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Interceptor intercepts database changes and enforces approval requirements for sensitive data.
type Interceptor struct {
	manager *Manager
	store   *store.Store
}

// NewInterceptor creates a new sensitive data interceptor.
func NewInterceptor(manager *Manager, store *store.Store) *Interceptor {
	return &Interceptor{
		manager: manager,
		store:   store,
	}
}

// InterceptChange intercepts a database change and checks if approval is required.
func (i *Interceptor) InterceptChange(ctx context.Context, change *DatabaseChange) (*ApprovalRequirement, error) {
	// Classify sensitive data in the change
	sensitiveFields, err := i.classifyChangeSensitivity(ctx, change)
	if err != nil {
		return nil, errors.Wrap(err, "failed to classify change sensitivity")
	}

	if len(sensitiveFields) == 0 {
		// No sensitive data, no approval required
		return &ApprovalRequirement{
			Required:          false,
			SensitivityLevel:  storepb.SensitiveDataLevel_LOW,
			ApprovalFlow:      nil,
			SensitiveFields:   []string{},
		}, nil
	}

	// Determine the highest sensitivity level
	highestLevel := storepb.SensitiveDataLevel_LOW
	for _, field := range sensitiveFields {
		if field.Level > highestLevel {
			highestLevel = field.Level
		}
	}

	// Get the approval flow for the highest sensitivity level
	flow, err := i.manager.GetApprovalFlowForLevel(ctx, highestLevel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get approval flow")
	}

	// Check if approval is required
	required := highestLevel != storepb.SensitiveDataLevel_LOW && flow != nil && len(flow.Stages) > 0

	// Collect sensitive field names
	fieldNames := []string{}
	for _, field := range sensitiveFields {
		fieldNames = append(fieldNames, fmt.Sprintf("%s.%s", field.TableName, field.FieldName))
	}

	return &ApprovalRequirement{
		Required:          required,
		SensitivityLevel:  highestLevel,
		ApprovalFlow:      flow,
		SensitiveFields:   fieldNames,
	}, nil
}

// CheckApprovalStatus checks if a change has been approved.
func (i *Interceptor) CheckApprovalStatus(ctx context.Context, issueID int32) (*ApprovalStatus, error) {
	progress, err := i.manager.EvaluateApprovalProgress(ctx, issueID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate approval progress")
	}

	if progress.IsRejected {
		return &ApprovalStatus{
			Approved:    false,
			Reason:      "Change was rejected",
			Progress:    progress,
		}, nil
	}

	if !progress.IsFullyApproved {
		return &ApprovalStatus{
			Approved:    false,
			Reason:      fmt.Sprintf("Awaiting approval at stage %d of %d", progress.CurrentStage, progress.TotalStages),
			Progress:    progress,
		}, nil
	}

	return &ApprovalStatus{
		Approved:    true,
		Reason:      "Change has been fully approved",
		Progress:    progress,
	}, nil
}

// LogChange logs a database change to the audit log.
func (i *Interceptor) LogChange(ctx context.Context, change *DatabaseChange, status *ApprovalStatus) error {
	// TODO: Implement audit log integration
	// This should create an entry in the existing audit log system
	return nil
}

// classifyChangeSensitivity classifies the sensitivity of fields in a database change.
func (i *Interceptor) classifyChangeSensitivity(ctx context.Context, change *DatabaseChange) ([]*SensitiveField, error) {
	var sensitiveFields []*SensitiveField

	for _, tableChange := range change.TableChanges {
		for _, fieldChange := range tableChange.FieldChanges {
			level, err := i.manager.ClassifySensitiveData(
				ctx,
				change.InstanceID,
				change.DatabaseName,
				tableChange.TableName,
				fieldChange.FieldName,
				fieldChange.DataType,
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to classify field %s.%s", tableChange.TableName, fieldChange.FieldName)
			}

			if level != storepb.SensitiveDataLevel_LOW {
				sensitiveFields = append(sensitiveFields, &SensitiveField{
					TableName:  tableChange.TableName,
					FieldName:  fieldChange.FieldName,
					Level:      level,
				})
			}
		}
	}

	return sensitiveFields, nil
}

// DatabaseChange represents a database change.
type DatabaseChange struct {
	InstanceID    int32
	DatabaseName  string
	TableChanges  []*TableChange
	SQL           string
	IssueID       int32
	UserID        int32
}

// TableChange represents a change to a database table.
type TableChange struct {
	TableName     string
	FieldChanges  []*FieldChange
}

// FieldChange represents a change to a database field.
type FieldChange struct {
	FieldName     string
	DataType      string
	OldValue      string
	NewValue      string
	ChangeType    FieldChangeType
}

// FieldChangeType represents the type of field change.
type FieldChangeType string

const (
	FieldChangeTypeCreate FieldChangeType = "CREATE"
	FieldChangeTypeUpdate FieldChangeType = "UPDATE"
	FieldChangeTypeDelete FieldChangeType = "DELETE"
)

// SensitiveField represents a sensitive field in a database change.
type SensitiveField struct {
	TableName  string
	FieldName  string
	Level      storepb.SensitiveDataLevel
}

// ApprovalRequirement represents the approval requirement for a change.
type ApprovalRequirement struct {
	Required          bool
	SensitivityLevel  storepb.SensitiveDataLevel
	ApprovalFlow      *storepb.ApprovalFlow
	SensitiveFields   []string
}

// ApprovalStatus represents the approval status of a change.
type ApprovalStatus struct {
	Approved    bool
	Reason      string
	Progress    *ApprovalProgress
}
