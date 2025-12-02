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

// ApprovalFlowMessage represents an approval flow configuration.
type ApprovalFlowMessage struct {
	ID               string
	DisplayName      string
	Description      string
	SensitivityLevel int32
	Steps            []*ApprovalStepMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ApprovalStepMessage represents an approval step.
type ApprovalStepMessage struct {
	Name         string
	Description  string
	Role         string
	Order        int32
	MinApprovals int32
	MaxApprovals int32
}

// ApprovalRequestMessage represents an approval request.
type ApprovalRequestMessage struct {
	ID               string
	Title            string
	Description      string
	IssueID          string
	SensitivityLevel int32
	ApprovalFlowID   string
	Status           int32
	Submitter        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FindApprovalFlowMessage represents the message to find approval flows.
type FindApprovalFlowMessage struct {
	ID               *string
	SensitivityLevel *int32
	Limit            *int
}

// FindApprovalRequestMessage represents the message to find approval requests.
type FindApprovalRequestMessage struct {
	ID               *string
	IssueID          *string
	SensitivityLevel *int32
	Status           *int32
	Submitter        *string
	Limit            *int
}

// CreateApprovalFlow creates a new approval flow.
func (s *Store) CreateApprovalFlow(ctx context.Context, af *ApprovalFlowMessage) (*ApprovalFlowMessage, error) {
	// Generate ID
	af.ID = fmt.Sprintf("af_%d", time.Now().UnixNano())
	af.CreatedAt = time.Now()
	af.UpdatedAt = time.Now()

	// Insert into database
	query := `
		INSERT INTO approval_flows (id, display_name, description, sensitivity_level, steps, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Convert steps to JSON
	stepsJSON, err := s.convertStepsToJSON(af.Steps)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert steps to JSON")
	}

	_, err = s.dbConnManager.ExecContext(ctx, query,
		af.ID,
		af.DisplayName,
		af.Description,
		af.SensitivityLevel,
		stepsJSON,
		af.CreatedAt,
		af.UpdatedAt,
	)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create approval flow")
	}

	return af, nil
}

// GetApprovalFlow gets an approval flow by ID.
func (s *Store) GetApprovalFlow(ctx context.Context, find *FindApprovalFlowMessage) (*ApprovalFlowMessage, error) {
	if find.ID == nil {
		return nil, errors.New("ID is required")
	}

	query := `
		SELECT id, display_name, description, sensitivity_level, steps, created_at, updated_at
		FROM approval_flows
		WHERE id = $1
	`

	row := s.dbConnManager.QueryRowContext(ctx, query, *find.ID)

	var af ApprovalFlowMessage
	var stepsJSON string

	err := row.Scan(
		&af.ID,
		&af.DisplayName,
		&af.Description,
		&af.SensitivityLevel,
		&stepsJSON,
		&af.CreatedAt,
		&af.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get approval flow")
	}

	// Convert steps from JSON
	af.Steps, err = s.convertStepsFromJSON(stepsJSON)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert steps from JSON")
	}

	return &af, nil
}

// ListApprovalFlows lists approval flows.
func (s *Store) ListApprovalFlows(ctx context.Context, find *FindApprovalFlowMessage) ([]*ApprovalFlowMessage, error) {
	query := `
		SELECT id, display_name, description, sensitivity_level, steps, created_at, updated_at
		FROM approval_flows
		WHERE 1=1
	`

	var args []interface{}
	argCount := 1

	// Apply filters
	if find.SensitivityLevel != nil {
		query += fmt.Sprintf(" AND sensitivity_level = $%d", argCount)
		args = append(args, *find.SensitivityLevel)
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
		return nil, errors.Wrapf(err, "failed to list approval flows")
	}
	defer rows.Close()

	var approvalFlows []*ApprovalFlowMessage

	for rows.Next() {
		var af ApprovalFlowMessage
		var stepsJSON string

		err := rows.Scan(
			&af.ID,
			&af.DisplayName,
			&af.Description,
			&af.SensitivityLevel,
			&stepsJSON,
			&af.CreatedAt,
			&af.UpdatedAt,
		)

		if err != nil {
			return nil, errors.Wrapf(err, "failed to scan approval flow")
		}

		// Convert steps from JSON
		af.Steps, err = s.convertStepsFromJSON(stepsJSON)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert steps from JSON")
		}

		approvalFlows = append(approvalFlows, &af)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate approval flows")
	}

	return approvalFlows, nil
}

// UpdateApprovalFlow updates an approval flow.
func (s *Store) UpdateApprovalFlow(ctx context.Context, id string, update *UpdateApprovalFlowMessage) (*ApprovalFlowMessage, error) {
	query := `
		UPDATE approval_flows
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

	if update.SensitivityLevel != nil {
		query += fmt.Sprintf(", sensitivity_level = $%d", argCount)
		args = append(args, *update.SensitivityLevel)
		argCount++
	}

	if update.Steps != nil {
		stepsJSON, err := s.convertStepsToJSON(update.Steps)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert steps to JSON")
		}
		query += fmt.Sprintf(", steps = $%d", argCount)
		args = append(args, stepsJSON)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)
	argCount++

	result, err := s.dbConnManager.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update approval flow")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return nil, errors.New("approval flow not found")
	}

	// Get the updated approval flow
	return s.GetApprovalFlow(ctx, &FindApprovalFlowMessage{ID: &id})
}

// DeleteApprovalFlow deletes an approval flow.
func (s *Store) DeleteApprovalFlow(ctx context.Context, id string) error {
	query := `
		DELETE FROM approval_flows
		WHERE id = $1
	`

	result, err := s.dbConnManager.ExecContext(ctx, query, id)
	if err != nil {
		return errors.Wrapf(err, "failed to delete approval flow")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("approval flow not found")
	}

	return nil
}

// CreateApprovalRequest creates a new approval request.
func (s *Store) CreateApprovalRequest(ctx context.Context, ar *ApprovalRequestMessage) (*ApprovalRequestMessage, error) {
	// Generate ID
	ar.ID = fmt.Sprintf("ar_%d", time.Now().UnixNano())
	ar.CreatedAt = time.Now()
	ar.UpdatedAt = time.Now()

	// Insert into database
	query := `
		INSERT INTO approval_requests (id, title, description, issue_id, sensitivity_level, approval_flow_id, status, submitter, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.dbConnManager.ExecContext(ctx, query,
		ar.ID,
		ar.Title,
		ar.Description,
		ar.IssueID,
		ar.SensitivityLevel,
		ar.ApprovalFlowID,
		ar.Status,
		ar.Submitter,
		ar.CreatedAt,
		ar.UpdatedAt,
	)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create approval request")
	}

	return ar, nil
}

// GetApprovalRequest gets an approval request by ID.
func (s *Store) GetApprovalRequest(ctx context.Context, find *FindApprovalRequestMessage) (*ApprovalRequestMessage, error) {
	if find.ID == nil {
		return nil, errors.New("ID is required")
	}

	query := `
		SELECT id, title, description, issue_id, sensitivity_level, approval_flow_id, status, submitter, created_at, updated_at
		FROM approval_requests
		WHERE id = $1
	`

	row := s.dbConnManager.QueryRowContext(ctx, query, *find.ID)

	var ar ApprovalRequestMessage

	err := row.Scan(
		&ar.ID,
		&ar.Title,
		&ar.Description,
		&ar.IssueID,
		&ar.SensitivityLevel,
		&ar.ApprovalFlowID,
		&ar.Status,
		&ar.Submitter,
		&ar.CreatedAt,
		&ar.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get approval request")
	}

	return &ar, nil
}

// ListApprovalRequests lists approval requests.
func (s *Store) ListApprovalRequests(ctx context.Context, find *FindApprovalRequestMessage) ([]*ApprovalRequestMessage, error) {
	query := `
		SELECT id, title, description, issue_id, sensitivity_level, approval_flow_id, status, submitter, created_at, updated_at
		FROM approval_requests
		WHERE 1=1
	`

	var args []interface{}
	argCount := 1

	// Apply filters
	if find.IssueID != nil {
		query += fmt.Sprintf(" AND issue_id = $%d", argCount)
		args = append(args, *find.IssueID)
		argCount++
	}

	if find.SensitivityLevel != nil {
		query += fmt.Sprintf(" AND sensitivity_level = $%d", argCount)
		args = append(args, *find.SensitivityLevel)
		argCount++
	}

	if find.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *find.Status)
		argCount++
	}

	if find.Submitter != nil {
		query += fmt.Sprintf(" AND submitter = $%d", argCount)
		args = append(args, *find.Submitter)
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
		return nil, errors.Wrapf(err, "failed to list approval requests")
	}
	defer rows.Close()

	var approvalRequests []*ApprovalRequestMessage

	for rows.Next() {
		var ar ApprovalRequestMessage

		err := rows.Scan(
			&ar.ID,
			&ar.Title,
			&ar.Description,
			&ar.IssueID,
			&ar.SensitivityLevel,
			&ar.ApprovalFlowID,
			&ar.Status,
			&ar.Submitter,
			&ar.CreatedAt,
			&ar.UpdatedAt,
		)

		if err != nil {
			return nil, errors.Wrapf(err, "failed to scan approval request")
		}

		approvalRequests = append(approvalRequests, &ar)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate approval requests")
	}

	return approvalRequests, nil
}

// UpdateApprovalRequest updates an approval request.
func (s *Store) UpdateApprovalRequest(ctx context.Context, id string, update *UpdateApprovalRequestMessage) (*ApprovalRequestMessage, error) {
	query := `
		UPDATE approval_requests
		SET updated_at = NOW()
	`

	var args []interface{}
	argCount := 1

	// Apply updates
	if update.Status != nil {
		query += fmt.Sprintf(", status = $%d", argCount)
		args = append(args, *update.Status)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)
	argCount++

	result, err := s.dbConnManager.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update approval request")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return nil, errors.New("approval request not found")
	}

	// Get the updated approval request
	return s.GetApprovalRequest(ctx, &FindApprovalRequestMessage{ID: &id})
}

// convertStepsToJSON converts steps to JSON string.
func (s *Store) convertStepsToJSON(steps []*ApprovalStepMessage) (string, error) {
	// Implementation depends on the database type and JSON support
	// For simplicity, we'll use a basic JSON format
	var jsonParts []string
	for _, step := range steps {
		jsonParts = append(jsonParts, fmt.Sprintf(`{"name":"%s","description":"%s","role":"%s","order":%d,"minApprovals":%d,"maxApprovals":%d}`,
			step.Name,
			step.Description,
			step.Role,
			step.Order,
			step.MinApprovals,
			step.MaxApprovals,
		))
	}
	return fmt.Sprintf("[%s]", strings.Join(jsonParts, ",")), nil
}

// convertStepsFromJSON converts JSON string to steps.
func (s *Store) convertStepsFromJSON(jsonStr string) ([]*ApprovalStepMessage, error) {
	// Implementation depends on the database type and JSON support
	// For simplicity, we'll return an empty slice
	// In a real implementation, you would parse the JSON and return the actual steps
	return []*ApprovalStepMessage{}, nil
}

// UpdateApprovalFlowMessage represents the message to update an approval flow.
type UpdateApprovalFlowMessage struct {
	DisplayName      *string
	Description      *string
	SensitivityLevel *int32
	Steps            []*ApprovalStepMessage
}

// UpdateApprovalRequestMessage represents the message to update an approval request.
type UpdateApprovalRequestMessage struct {
	Status *int32
}