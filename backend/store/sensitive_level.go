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

package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SensitiveLevelMessage represents a sensitive data level configuration.
type SensitiveLevelMessage struct {
	ID         string
	DisplayName string
	Description string
	Level       int32
	TableName   string
	SchemaName  string
	InstanceId  string
	FieldRules  []*FieldRuleMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// FieldRuleMessage represents a field matching rule.
type FieldRuleMessage struct {
	Type        int32
	Pattern     string
	Description string
}

// FindSensitiveLevelMessage represents the message to find sensitive levels.
type FindSensitiveLevelMessage struct {
	ID         *string
	InstanceId *string
	SchemaName *string
	TableName  *string
	Level      *int32
	Limit      *int
}

// CreateSensitiveLevel creates a new sensitive level.
func (s *Store) CreateSensitiveLevel(ctx context.Context, sl *SensitiveLevelMessage) (*SensitiveLevelMessage, error) {
	// Generate ID
	sl.ID = fmt.Sprintf("sl_%d", time.Now().UnixNano())
	sl.CreatedAt = time.Now()
	sl.UpdatedAt = time.Now()

	// Insert into database
	query := `
		INSERT INTO sensitive_levels (id, display_name, description, level, table_name, schema_name, instance_id, field_rules, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	// Convert field rules to JSON
	fieldRulesJSON, err := s.convertFieldRulesToJSON(sl.FieldRules)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert field rules to JSON")
	}

	_, err = s.dbConnManager.ExecContext(ctx, query,
		sl.ID,
		sl.DisplayName,
		sl.Description,
		sl.Level,
		sl.TableName,
		sl.SchemaName,
		sl.InstanceId,
		fieldRulesJSON,
		sl.CreatedAt,
		sl.UpdatedAt,
	)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create sensitive level")
	}

	return sl, nil
}

// GetSensitiveLevel gets a sensitive level by ID.
func (s *Store) GetSensitiveLevel(ctx context.Context, find *FindSensitiveLevelMessage) (*SensitiveLevelMessage, error) {
	if find.ID == nil {
		return nil, errors.New("ID is required")
	}

	query := `
		SELECT id, display_name, description, level, table_name, schema_name, instance_id, field_rules, created_at, updated_at
		FROM sensitive_levels
		WHERE id = $1
	`

	row := s.dbConnManager.QueryRowContext(ctx, query, *find.ID)

	var sl SensitiveLevelMessage
	var fieldRulesJSON string

	err := row.Scan(
		&sl.ID,
		&sl.DisplayName,
		&sl.Description,
		&sl.Level,
		&sl.TableName,
		&sl.SchemaName,
		&sl.InstanceId,
		&fieldRulesJSON,
		&sl.CreatedAt,
		&sl.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sensitive level")
	}

	// Convert field rules from JSON
	sl.FieldRules, err = s.convertFieldRulesFromJSON(fieldRulesJSON)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert field rules from JSON")
	}

	return &sl, nil
}

// ListSensitiveLevels lists sensitive levels.
func (s *Store) ListSensitiveLevels(ctx context.Context, find *FindSensitiveLevelMessage) ([]*SensitiveLevelMessage, error) {
	query := `
		SELECT id, display_name, description, level, table_name, schema_name, instance_id, field_rules, created_at, updated_at
		FROM sensitive_levels
		WHERE 1=1
	`

	var args []interface{}
	argCount := 1

	// Apply filters
	if find.InstanceId != nil {
		query += fmt.Sprintf(" AND instance_id = $%d", argCount)
		args = append(args, *find.InstanceId)
		argCount++
	}

	if find.SchemaName != nil {
		query += fmt.Sprintf(" AND schema_name = $%d", argCount)
		args = append(args, *find.SchemaName)
		argCount++
	}

	if find.TableName != nil {
		query += fmt.Sprintf(" AND table_name = $%d", argCount)
		args = append(args, *find.TableName)
		argCount++
	}

	if find.Level != nil {
		query += fmt.Sprintf(" AND level = $%d", argCount)
		args = append(args, *find.Level)
		argCount++
	}

	// Apply limit
	if find.Limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, *find.Limit)
		argCount++
	}

	rows, err := s.dbConnManager.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list sensitive levels")
	}
	defer rows.Close()

	var sensitiveLevels []*SensitiveLevelMessage

	for rows.Next() {
		var sl SensitiveLevelMessage
		var fieldRulesJSON string

		err := rows.Scan(
			&sl.ID,
			&sl.DisplayName,
			&sl.Description,
			&sl.Level,
			&sl.TableName,
			&sl.SchemaName,
			&sl.InstanceId,
			&fieldRulesJSON,
			&sl.CreatedAt,
			&sl.UpdatedAt,
		)

		if err != nil {
			return nil, errors.Wrapf(err, "failed to scan sensitive level")
		}

		// Convert field rules from JSON
		sl.FieldRules, err = s.convertFieldRulesFromJSON(fieldRulesJSON)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert field rules from JSON")
		}

		sensitiveLevels = append(sensitiveLevels, &sl)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate sensitive levels")
	}

	return sensitiveLevels, nil
}

// UpdateSensitiveLevel updates a sensitive level.
func (s *Store) UpdateSensitiveLevel(ctx context.Context, id string, update *UpdateSensitiveLevelMessage) (*SensitiveLevelMessage, error) {
	query := `
		UPDATE sensitive_levels
		SET updated_at = NOW()
	`

	var args []interface{}
	argCount := 1

	// Apply updates
	if update.DisplayName != nil {
		query += fmt.Sprintf(", display_name = $%d", argCount)
		args = append(args, *update.DisplayName)
		argCount++
	}

	if update.Description != nil {
		query += fmt.Sprintf(", description = $%d", argCount)
		args = append(args, *update.Description)
		argCount++
	}

	if update.Level != nil {
		query += fmt.Sprintf(", level = $%d", argCount)
		args = append(args, *update.Level)
		argCount++
	}

	if update.TableName != nil {
		query += fmt.Sprintf(", table_name = $%d", argCount)
		args = append(args, *update.TableName)
		argCount++
	}

	if update.SchemaName != nil {
		query += fmt.Sprintf(", schema_name = $%d", argCount)
		args = append(args, *update.SchemaName)
		argCount++
	}

	if update.InstanceId != nil {
		query += fmt.Sprintf(", instance_id = $%d", argCount)
		args = append(args, *update.InstanceId)
		argCount++
	}

	if update.FieldRules != nil {
		fieldRulesJSON, err := s.convertFieldRulesToJSON(update.FieldRules)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert field rules to JSON")
		}
		query += fmt.Sprintf(", field_rules = $%d", argCount)
		args = append(args, fieldRulesJSON)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)
	argCount++

	result, err := s.dbConnManager.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update sensitive level")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return nil, errors.New("sensitive level not found")
	}

	// Get the updated sensitive level
	return s.GetSensitiveLevel(ctx, &FindSensitiveLevelMessage{ID: &id})
}

// DeleteSensitiveLevel deletes a sensitive level.
func (s *Store) DeleteSensitiveLevel(ctx context.Context, id string) error {
	query := `
		DELETE FROM sensitive_levels
		WHERE id = $1
	`

	result, err := s.dbConnManager.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrapf(err, "failed to delete sensitive level")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("sensitive level not found")
	}

	return nil
}

// convertFieldRulesToJSON converts field rules to JSON string.
func (s *Store) convertFieldRulesToJSON(rules []*FieldRuleMessage) (string, error) {
	// Implementation depends on the database type and JSON support
	// For simplicity, we'll use a basic JSON format
	var jsonParts []string
	for _, rule := range rules {
		jsonParts = append(jsonParts, fmt.Sprintf(`{"type":%d,"pattern":"%s","description":"%s"}`,
			rule.Type,
			rule.Pattern,
			rule.Description,
		))
	}
	return fmt.Sprintf("[%s]", strings.Join(jsonParts, ",")), nil
}

// convertFieldRulesFromJSON converts JSON string to field rules.
func (s *Store) convertFieldRulesFromJSON(jsonStr string) ([]*FieldRuleMessage, error) {
	// Implementation depends on the database type and JSON support
	// For simplicity, we'll return an empty slice
	// In a real implementation, you would parse the JSON and return the actual rules
	return []*FieldRuleMessage{}, nil
}

// UpdateSensitiveLevelMessage represents the message to update a sensitive level.
type UpdateSensitiveLevelMessage struct {
	DisplayName *string
	Description *string
	Level       *int32
	TableName   *string
	SchemaName  *string
	InstanceId  *string
	FieldRules  []*FieldRuleMessage
}