package sensitive_data

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Manager is the manager for sensitive data classification and approval flow.
type Manager struct {
	store *store.Store
}

// NewManager creates a new sensitive data manager.
func NewManager(store *store.Store) *Manager {
	return &Manager{
		store: store,
	}
}

// ClassifySensitiveData classifies the sensitivity level of a database field.
func (m *Manager) ClassifySensitiveData(ctx context.Context, instanceID int32, databaseName, tableName, fieldName, dataType string) (storepb.SensitiveDataLevel, error) {
	// First check exact mappings
	mappings, err := m.store.ListSensitiveDataMappings(ctx, &store.FindSensitiveDataMappingMessage{
		InstanceID:    instanceID,
		DatabaseName:  databaseName,
		TableName:     tableName,
		FieldName:     fieldName,
	})
	if err != nil {
		return storepb.SensitiveDataLevel_SENSITIVE_DATA_LEVEL_UNSPECIFIED, errors.Wrap(err, "failed to list sensitive data mappings")
	}

	if len(mappings) > 0 {
		// Return the highest sensitivity level if multiple mappings exist
		level := storepb.SensitiveDataLevel_LOW
		for _, mapping := range mappings {
			if mapping.Level > level {
				level = mapping.Level
			}
		}
		return level, nil
	}

	// Check rules if no exact mappings
	rules, err := m.store.ListSensitiveDataRules(ctx, &store.FindSensitiveDataRuleMessage{
		Enabled: true,
	})
	if err != nil {
		return storepb.SensitiveDataLevel_SENSITIVE_DATA_LEVEL_UNSPECIFIED, errors.Wrap(err, "failed to list sensitive data rules")
	}

	for _, rule := range rules {
		if m.matchesRule(rule, tableName, fieldName, dataType) {
			return rule.Level, nil
		}
	}

	// Default to low sensitivity if no rules match
	return storepb.SensitiveDataLevel_LOW, nil
}

// matchesRule checks if a field matches a sensitive data rule.
func (m *Manager) matchesRule(rule *storepb.SensitiveDataRule, tableName, fieldName, dataType string) bool {
	// Check table pattern
	if rule.TablePattern != "" && !matchesPattern(rule.TablePattern, tableName) {
		return false
	}

	// Check field pattern
	if rule.FieldPattern != "" && !matchesPattern(rule.FieldPattern, fieldName) {
		return false
	}

	// Check data type
	if rule.DataType != "" && strings.ToLower(rule.DataType) != strings.ToLower(dataType) {
		return false
	}

	return true
}

// matchesPattern checks if a string matches a pattern (supports wildcard *).
func matchesPattern(pattern, value string) bool {
	if pattern == "" {
		return true
	}

	// Convert wildcard pattern to regex
	regexPattern := strings.ReplaceAll(pattern, "*", ".*")
	regexPattern = fmt.Sprintf("^%s$", regexPattern)
	matched, _ := regexp.MatchString(regexPattern, value)
	return matched
}

// GetApprovalFlowForLevel gets the approval flow for a sensitivity level.
func (m *Manager) GetApprovalFlowForLevel(ctx context.Context, level storepb.SensitiveDataLevel) (*storepb.ApprovalFlow, error) {
	templates, err := m.store.ListApprovalTemplates(ctx, &store.FindApprovalTemplateMessage{
		Level: level,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list approval templates")
	}

	if len(templates) == 0 {
		// Create default template if none exists
		return m.createDefaultApprovalFlow(level)
	}

	// Return the first matching template
	return templates[0].Flow, nil
}

// createDefaultApprovalFlow creates a default approval flow for a sensitivity level.
func (m *Manager) createDefaultApprovalFlow(level storepb.SensitiveDataLevel) (*storepb.ApprovalFlow, error) {
	var stages []*storepb.ApprovalStage

	switch level {
	case storepb.SensitiveDataLevel_HIGH:
		// High sensitivity: 2-level approval (department head → DBA)
		stages = []*storepb.ApprovalStage{
			{
				Id:              1,
				Name:            "Department Head Approval",
				Description:     "Approval from department head required",
				Roles:           []string{"department_head"},
				RequiredApprovals: 1,
				TimeoutSeconds:  86400, // 24 hours
			},
			{
				Id:              2,
				Name:            "DBA Approval",
				Description:     "Approval from DBA required",
				Roles:           []string{"dba"},
				RequiredApprovals: 1,
				TimeoutSeconds:  86400, // 24 hours
			},
		}

	case storepb.SensitiveDataLevel_MEDIUM:
		// Medium sensitivity: 1-level approval (DBA)
		stages = []*storepb.ApprovalStage{
			{
				Id:              1,
				Name:            "DBA Approval",
				Description:     "Approval from DBA required",
				Roles:           []string{"dba"},
				RequiredApprovals: 1,
				TimeoutSeconds:  86400, // 24 hours
			},
		}

	case storepb.SensitiveDataLevel_LOW:
		// Low sensitivity: no approval required
		stages = []*storepb.ApprovalStage{}

	default:
		// Default to low sensitivity
		stages = []*storepb.ApprovalStage{}
	}

	return &storepb.ApprovalFlow{
		Stages: stages,
		Enabled: true,
	}, nil
}

// EvaluateApprovalProgress evaluates the approval progress of an issue.
func (m *Manager) EvaluateApprovalProgress(ctx context.Context, issueID int32) (*ApprovalProgress, error) {
	issue, err := m.store.GetIssue(ctx, issueID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issue")
	}

	if issue.Payload.Approval == nil || issue.Payload.Approval.ApprovalTemplate == nil {
		return &ApprovalProgress{
			CurrentStage:    0,
			TotalStages:     0,
			IsFullyApproved: true,
			IsRejected:      false,
		}, nil
	}

	flow := issue.Payload.Approval.ApprovalTemplate.Flow
	if flow == nil || len(flow.Stages) == 0 {
		return &ApprovalProgress{
			CurrentStage:    0,
			TotalStages:     0,
			IsFullyApproved: true,
			IsRejected:      false,
		}, nil
	}

	// Check if any stage is rejected
	for _, approver := range issue.Payload.Approval.Approvers {
		if approver.Status == storepb.IssuePayloadApproval_Approver_REJECTED {
			return &ApprovalProgress{
				CurrentStage:    approver.StageId,
				TotalStages:     len(flow.Stages),
				IsFullyApproved: false,
				IsRejected:      true,
			}, nil
		}
	}

	// Find current stage
	currentStage := 0
	for i, stage := range flow.Stages {
		stageApproved := true
		for _, approver := range issue.Payload.Approval.Approvers {
			if approver.StageId == stage.Id && approver.Status != storepb.IssuePayloadApproval_Approver_APPROVED {
				stageApproved = false
				break
			}
		}

		if !stageApproved {
			currentStage = stage.Id
			break
		}
	}

	// If all stages are approved
	if currentStage == 0 {
		return &ApprovalProgress{
			CurrentStage:    len(flow.Stages),
			TotalStages:     len(flow.Stages),
			IsFullyApproved: true,
			IsRejected:      false,
		}, nil
	}

	return &ApprovalProgress{
		CurrentStage:    currentStage,
		TotalStages:     len(flow.Stages),
		IsFullyApproved: false,
		IsRejected:      false,
	}, nil
}

// ApprovalProgress represents the approval progress of an issue.
type ApprovalProgress struct {
	CurrentStage    int32
	TotalStages     int32
	IsFullyApproved bool
	IsRejected      bool
}
