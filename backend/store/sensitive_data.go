package store

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// FindSensitiveDataRuleMessage is the message for finding sensitive data rules.
type FindSensitiveDataRuleMessage struct {
	ID      *int32
	Level   *storepb.SensitiveDataLevel
	Enabled *bool
}

// ListSensitiveDataRules lists sensitive data rules.
func (s *Store) ListSensitiveDataRules(ctx context.Context, find *FindSensitiveDataRuleMessage) ([]*storepb.SensitiveDataRule, error) {
	// TODO: Implement database query
	return []*storepb.SensitiveDataRule{}, nil
}

// GetSensitiveDataRule gets a sensitive data rule by ID.
func (s *Store) GetSensitiveDataRule(ctx context.Context, id int32) (*storepb.SensitiveDataRule, error) {
	// TODO: Implement database query
	return nil, nil
}

// CreateSensitiveDataRule creates a sensitive data rule.
func (s *Store) CreateSensitiveDataRule(ctx context.Context, rule *storepb.SensitiveDataRule) (*storepb.SensitiveDataRule, error) {
	// TODO: Implement database insert
	return rule, nil
}

// UpdateSensitiveDataRule updates a sensitive data rule.
func (s *Store) UpdateSensitiveDataRule(ctx context.Context, rule *storepb.SensitiveDataRule) (*storepb.SensitiveDataRule, error) {
	// TODO: Implement database update
	return rule, nil
}

// DeleteSensitiveDataRule deletes a sensitive data rule.
func (s *Store) DeleteSensitiveDataRule(ctx context.Context, id int32) error {
	// TODO: Implement database delete
	return nil
}

// FindSensitiveDataMappingMessage is the message for finding sensitive data mappings.
type FindSensitiveDataMappingMessage struct {
	InstanceID    *int32
	DatabaseName  *string
	TableName     *string
	FieldName     *string
	Level         *storepb.SensitiveDataLevel
}

// ListSensitiveDataMappings lists sensitive data mappings.
func (s *Store) ListSensitiveDataMappings(ctx context.Context, find *FindSensitiveDataMappingMessage) ([]*storepb.SensitiveDataMapping, error) {
	// TODO: Implement database query
	return []*storepb.SensitiveDataMapping{}, nil
}

// GetSensitiveDataMapping gets a sensitive data mapping by ID.
func (s *Store) GetSensitiveDataMapping(ctx context.Context, id int32) (*storepb.SensitiveDataMapping, error) {
	// TODO: Implement database query
	return nil, nil
}

// CreateSensitiveDataMapping creates a sensitive data mapping.
func (s *Store) CreateSensitiveDataMapping(ctx context.Context, mapping *storepb.SensitiveDataMapping) (*storepb.SensitiveDataMapping, error) {
	// TODO: Implement database insert
	return mapping, nil
}

// UpdateSensitiveDataMapping updates a sensitive data mapping.
func (s *Store) UpdateSensitiveDataMapping(ctx context.Context, mapping *storepb.SensitiveDataMapping) (*storepb.SensitiveDataMapping, error) {
	// TODO: Implement database update
	return mapping, nil
}

// DeleteSensitiveDataMapping deletes a sensitive data mapping.
func (s *Store) DeleteSensitiveDataMapping(ctx context.Context, id int32) error {
	// TODO: Implement database delete
	return nil
}

// FindApprovalTemplateMessage is the message for finding approval templates.
type FindApprovalTemplateMessage struct {
	ID    *int32
	Level *storepb.SensitiveDataLevel
}

// ListApprovalTemplates lists approval templates.
func (s *Store) ListApprovalTemplates(ctx context.Context, find *FindApprovalTemplateMessage) ([]*storepb.ApprovalTemplate, error) {
	// TODO: Implement database query
	return []*storepb.ApprovalTemplate{}, nil
}

// GetApprovalTemplate gets an approval template by ID.
func (s *Store) GetApprovalTemplate(ctx context.Context, id int32) (*storepb.ApprovalTemplate, error) {
	// TODO: Implement database query
	return nil, nil
}

// CreateApprovalTemplate creates an approval template.
func (s *Store) CreateApprovalTemplate(ctx context.Context, template *storepb.ApprovalTemplate) (*storepb.ApprovalTemplate, error) {
	// TODO: Implement database insert
	return template, nil
}

// UpdateApprovalTemplate updates an approval template.
func (s *Store) UpdateApprovalTemplate(ctx context.Context, template *storepb.ApprovalTemplate) (*storepb.ApprovalTemplate, error) {
	// TODO: Implement database update
	return template, nil
}

// DeleteApprovalTemplate deletes an approval template.
func (s *Store) DeleteApprovalTemplate(ctx context.Context, id int32) error {
	// TODO: Implement database delete
	return nil
}

// FindApprovalHistoryMessage is the message for finding approval histories.
type FindApprovalHistoryMessage struct {
	IssueID *int32
}

// ListApprovalHistories lists approval histories.
func (s *Store) ListApprovalHistories(ctx context.Context, find *FindApprovalHistoryMessage) ([]*storepb.ApprovalHistory, error) {
	// TODO: Implement database query
	return []*storepb.ApprovalHistory{}, nil
}

// CreateApprovalHistory creates an approval history.
func (s *Store) CreateApprovalHistory(ctx context.Context, history *storepb.ApprovalHistory) (*storepb.ApprovalHistory, error) {
	// TODO: Implement database insert
	return history, nil
}
