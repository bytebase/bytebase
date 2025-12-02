package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/vcs/git"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/store/pg"
	"github.com/bytebase/bytebase/common/log"
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// SensitiveApprovalStore is the store for sensitive levels and approval flows.
type SensitiveApprovalStore interface {
	// Sensitive Level APIs
	ListSensitiveLevels(ctx context.Context, req *v1.ListSensitiveLevelsRequest) ([]*v1.SensitiveLevel, error)
	GetSensitiveLevel(ctx context.Context, sensitiveLevelID string) (*v1.SensitiveLevel, error)
	CreateSensitiveLevel(ctx context.Context, sensitiveLevel *v1.SensitiveLevel) error
	UpdateSensitiveLevel(ctx context.Context, sensitiveLevel *v1.SensitiveLevel) error
	DeleteSensitiveLevel(ctx context.Context, sensitiveLevelID string) error

	// Approval Flow APIs
	ListApprovalFlows(ctx context.Context, req *v1.ListApprovalFlowsRequest) ([]*v1.ApprovalFlow, error)
	GetApprovalFlow(ctx context.Context, approvalFlowID string) (*v1.ApprovalFlow, error)
	CreateApprovalFlow(ctx context.Context, approvalFlow *v1.ApprovalFlow) error
	UpdateApprovalFlow(ctx context.Context, approvalFlow *v1.ApprovalFlow) error
	DeleteApprovalFlow(ctx context.Context, approvalFlowID string) error
}

// sensitiveApprovalStoreImpl is the implementation of SensitiveApprovalStore.
type sensitiveApprovalStoreImpl struct {
	*pg.PgDB
}

// NewSensitiveApprovalStore creates a new SensitiveApprovalStore.
func NewSensitiveApprovalStore(pgDB *pg.PgDB) SensitiveApprovalStore {
	return &sensitiveApprovalStoreImpl{
		PgDB: pgDB,
	}
}

// ListSensitiveLevels lists all sensitive levels.
func (s *sensitiveApprovalStoreImpl) ListSensitiveLevels(ctx context.Context, req *v1.ListSensitiveLevelsRequest) ([]*v1.SensitiveLevel, error) {
	var whereConditions []string
	var args []interface{}

	if req.FolderName != "" {
		whereConditions = append(whereConditions, "folder_name = $1")
		args = append(args, req.FolderName)
	}

	query := "SELECT id, name, display_name, severity, description, color, field_match_rules, create_time, update_time FROM sensitive_levels"
	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	query += " ORDER BY create_time DESC"

	rows, err := s.Query(ctx, query, args...)
	if err != nil {
		log.Errorf("Failed to query sensitive levels: %v", err)
		return nil, err
	}
	defer rows.Close()

	var sensitiveLevels []*v1.SensitiveLevel
	for rows.Next() {
		var sensitiveLevel v1.SensitiveLevel
		var fieldMatchRulesJSON string

		if err := rows.Scan(
			&sensitiveLevel.Name,
			&sensitiveLevel.DisplayName,
			&sensitiveLevel.Severity,
			&sensitiveLevel.Description,
			&sensitiveLevel.Color,
			&fieldMatchRulesJSON,
			&sensitiveLevel.CreateTime,
			&sensitiveLevel.UpdateTime,
		); err != nil {
			log.Errorf("Failed to scan sensitive level: %v", err)
			return nil, err
		}

		// Unmarshal field match rules
		if fieldMatchRulesJSON != "" {
			var fieldMatchRules []*v1.FieldMatchRule
			if err := json.Unmarshal([]byte(fieldMatchRulesJSON), &fieldMatchRules); err != nil {
				log.Errorf("Failed to unmarshal field match rules: %v", err)
				return nil, err
			}
			sensitiveLevel.FieldMatchRules = fieldMatchRules
		}

		sensitiveLevels = append(sensitiveLevels, &sensitiveLevel)
	}

	if err := rows.Err(); err != nil {
		log.Errorf("Failed to iterate sensitive levels: %v", err)
		return nil, err
	}

	return sensitiveLevels, nil
}

// GetSensitiveLevel gets a sensitive level by ID.
func (s *sensitiveApprovalStoreImpl) GetSensitiveLevel(ctx context.Context, sensitiveLevelID string) (*v1.SensitiveLevel, error) {
	var sensitiveLevel v1.SensitiveLevel
	var fieldMatchRulesJSON string

	query := "SELECT id, name, display_name, severity, description, color, field_match_rules, create_time, update_time FROM sensitive_levels WHERE id = $1"
	err := s.QueryRow(ctx, query, sensitiveLevelID).Scan(
		&sensitiveLevel.Name,
		&sensitiveLevel.DisplayName,
		&sensitiveLevel.Severity,
		&sensitiveLevel.Description,
		&sensitiveLevel.Color,
		&fieldMatchRulesJSON,
		&sensitiveLevel.CreateTime,
		&sensitiveLevel.UpdateTime,
	)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrNotFound
		}
		log.Errorf("Failed to get sensitive level: %v", err)
		return nil, err
	}

	// Unmarshal field match rules
	if fieldMatchRulesJSON != "" {
		var fieldMatchRules []*v1.FieldMatchRule
		if err := json.Unmarshal([]byte(fieldMatchRulesJSON), &fieldMatchRules); err != nil {
			log.Errorf("Failed to unmarshal field match rules: %v", err)
			return nil, err
		}
		sensitiveLevel.FieldMatchRules = fieldMatchRules
	}

	return &sensitiveLevel, nil
}

// CreateSensitiveLevel creates a new sensitive level.
func (s *sensitiveApprovalStoreImpl) CreateSensitiveLevel(ctx context.Context, sensitiveLevel *v1.SensitiveLevel) error {
	// Marshal field match rules to JSON
	fieldMatchRulesJSON, err := json.Marshal(sensitiveLevel.FieldMatchRules)
	if err != nil {
		log.Errorf("Failed to marshal field match rules: %v", err)
		return err
	}

	// Extract sensitive level ID from name
	parts := strings.Split(sensitiveLevel.Name, "/")
	if len(parts) != 2 {
		return errors.New("invalid sensitive level name format")
	}
	id := parts[1]

	query := "INSERT INTO sensitive_levels (id, name, display_name, severity, description, color, field_match_rules, create_time, update_time) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, err = s.Exec(ctx, query,
		id,
		sensitiveLevel.Name,
		sensitiveLevel.DisplayName,
		sensitiveLevel.Severity,
		sensitiveLevel.Description,
		sensitiveLevel.Color,
		string(fieldMatchRulesJSON),
		sensitiveLevel.CreateTime.AsTime(),
		sensitiveLevel.UpdateTime.AsTime(),
	)
	if err != nil {
		log.Errorf("Failed to create sensitive level: %v", err)
		return err
	}

	return nil
}

// UpdateSensitiveLevel updates an existing sensitive level.
func (s *sensitiveApprovalStoreImpl) UpdateSensitiveLevel(ctx context.Context, sensitiveLevel *v1.SensitiveLevel) error {
	// Marshal field match rules to JSON
	fieldMatchRulesJSON, err := json.Marshal(sensitiveLevel.FieldMatchRules)
	if err != nil {
		log.Errorf("Failed to marshal field match rules: %v", err)
		return err
	}

	// Extract sensitive level ID from name
	parts := strings.Split(sensitiveLevel.Name, "/")
	if len(parts) != 2 {
		return errors.New("invalid sensitive level name format")
	}
	id := parts[1]

	query := "UPDATE sensitive_levels SET display_name = $1, severity = $2, description = $3, color = $4, field_match_rules = $5, update_time = $6 WHERE id = $7"
	_, err = s.Exec(ctx, query,
		sensitiveLevel.DisplayName,
		sensitiveLevel.Severity,
		sensitiveLevel.Description,
		sensitiveLevel.Color,
		string(fieldMatchRulesJSON),
		sensitiveLevel.UpdateTime.AsTime(),
		id,
	)
	if err != nil {
		log.Errorf("Failed to update sensitive level: %v", err)
		return err
	}

	return nil
}

// DeleteSensitiveLevel deletes a sensitive level.
func (s *sensitiveApprovalStoreImpl) DeleteSensitiveLevel(ctx context.Context, sensitiveLevelID string) error {
	query := "DELETE FROM sensitive_levels WHERE id = $1"
	_, err := s.Exec(ctx, query, sensitiveLevelID)
	if err != nil {
		log.Errorf("Failed to delete sensitive level: %v", err)
		return err
	}

	return nil
}

// ListApprovalFlows lists all approval flows.
func (s *sensitiveApprovalStoreImpl) ListApprovalFlows(ctx context.Context, req *v1.ListApprovalFlowsRequest) ([]*v1.ApprovalFlow, error) {
	var whereConditions []string
	var args []interface{}

	if req.FolderName != "" {
		whereConditions = append(whereConditions, "folder_name = $1")
		args = append(args, req.FolderName)
	}

	query := "SELECT id, name, display_name, description, sensitive_severity, steps, create_time, update_time FROM approval_flows"
	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	query += " ORDER BY create_time DESC"

	rows, err := s.Query(ctx, query, args...)
	if err != nil {
		log.Errorf("Failed to query approval flows: %v", err)
		return nil, err
	}
	defer rows.Close()

	var approvalFlows []*v1.ApprovalFlow
	for rows.Next() {
		var approvalFlow v1.ApprovalFlow
		var stepsJSON string

		if err := rows.Scan(
			&approvalFlow.Name,
			&approvalFlow.DisplayName,
			&approvalFlow.Description,
			&approvalFlow.SensitiveSeverity,
			&stepsJSON,
			&approvalFlow.CreateTime,
			&approvalFlow.UpdateTime,
		); err != nil {
			log.Errorf("Failed to scan approval flow: %v", err)
			return nil, err
		}

		// Unmarshal steps
		if stepsJSON != "" {
			var steps []*v1.ApprovalStep
			if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
				log.Errorf("Failed to unmarshal steps: %v", err)
				return nil, err
			}
			approvalFlow.Steps = steps
		}

		approvalFlows = append(approvalFlows, &approvalFlow)
	}

	if err := rows.Err(); err != nil {
		log.Errorf("Failed to iterate approval flows: %v", err)
		return nil, err
	}

	return approvalFlows, nil
}

// GetApprovalFlow gets an approval flow by ID.
func (s *sensitiveApprovalStoreImpl) GetApprovalFlow(ctx context.Context, approvalFlowID string) (*v1.ApprovalFlow, error) {
	var approvalFlow v1.ApprovalFlow
	var stepsJSON string

	query := "SELECT id, name, display_name, description, sensitive_severity, steps, create_time, update_time FROM approval_flows WHERE id = $1"
	err := s.QueryRow(ctx, query, approvalFlowID).Scan(
		&approvalFlow.Name,
		&approvalFlow.DisplayName,
		&approvalFlow.Description,
		&approvalFlow.SensitiveSeverity,
		&stepsJSON,
		&approvalFlow.CreateTime,
		&approvalFlow.UpdateTime,
	)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrNotFound
		}
		log.Errorf("Failed to get approval flow: %v", err)
		return nil, err
	}

	// Unmarshal steps
	if stepsJSON != "" {
		var steps []*v1.ApprovalStep
		if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
			log.Errorf("Failed to unmarshal steps: %v", err)
			return nil, err
		}
		approvalFlow.Steps = steps
	}

	return &approvalFlow, nil
}

// CreateApprovalFlow creates a new approval flow.
func (s *sensitiveApprovalStoreImpl) CreateApprovalFlow(ctx context.Context, approvalFlow *v1.ApprovalFlow) error {
	// Marshal steps to JSON
	stepsJSON, err := json.Marshal(approvalFlow.Steps)
	if err != nil {
		log.Errorf("Failed to marshal steps: %v", err)
		return err
	}

	// Extract approval flow ID from name
	parts := strings.Split(approvalFlow.Name, "/")
	if len(parts) != 2 {
		return errors.New("invalid approval flow name format")
	}
	id := parts[1]

	query := "INSERT INTO approval_flows (id, name, display_name, description, sensitive_severity, steps, create_time, update_time) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
	_, err = s.Exec(ctx, query,
		id,
		approvalFlow.Name,
		approvalFlow.DisplayName,
		approvalFlow.Description,
		approvalFlow.SensitiveSeverity,
		string(stepsJSON),
		approvalFlow.CreateTime.AsTime(),
		approvalFlow.UpdateTime.AsTime(),
	)
	if err != nil {
		log.Errorf("Failed to create approval flow: %v", err)
		return err
	}

	return nil
}

// UpdateApprovalFlow updates an existing approval flow.
func (s *sensitiveApprovalStoreImpl) UpdateApprovalFlow(ctx context.Context, approvalFlow *v1.ApprovalFlow) error {
	// Marshal steps to JSON
	stepsJSON, err := json.Marshal(approvalFlow.Steps)
	if err != nil {
		log.Errorf("Failed to marshal steps: %v", err)
		return err
	}

	// Extract approval flow ID from name
	parts := strings.Split(approvalFlow.Name, "/")
	if len(parts) != 2 {
		return errors.New("invalid approval flow name format")
	}
	id := parts[1]

	query := "UPDATE approval_flows SET display_name = $1, description = $2, sensitive_severity = $3, steps = $4, update_time = $5 WHERE id = $6"
	_, err = s.Exec(ctx, query,
		approvalFlow.DisplayName,
		approvalFlow.Description,
		approvalFlow.SensitiveSeverity,
		string(stepsJSON),
		approvalFlow.UpdateTime.AsTime(),
		id,
	)
	if err != nil {
		log.Errorf("Failed to update approval flow: %v", err)
		return err
	}

	return nil
}

// DeleteApprovalFlow deletes an approval flow.
func (s *sensitiveApprovalStoreImpl) DeleteApprovalFlow(ctx context.Context, approvalFlowID string) error {
	query := "DELETE FROM approval_flows WHERE id = $1"
	_, err := s.Exec(ctx, query, approvalFlowID)
	if err != nil {
		log.Errorf("Failed to delete approval flow: %v", err)
		return err
	}

	return nil
}
