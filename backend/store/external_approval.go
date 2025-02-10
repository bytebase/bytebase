package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// ExternalApprovalMessage is the message for creating an external approval.
type ExternalApprovalMessage struct {
	// IssueUID is the unique identifier of the issue.
	IssueUID int
	// ApproverUID is the unique identifier of the approver.
	ApproverUID int
	// Type is the external approval type.
	Type api.ExternalApprovalType
	// Payload is the external approval payload.
	Payload string

	// Input Only fields.
	//
	// RequesterUID is the unique identifier of the requester.
	RequesterUID int

	// Output only fields.
	//
	// ID is the unique identifier of the external approval.
	ID int
}

// ListExternalApprovalMessage is the message for listing external approvals.
type ListExternalApprovalMessage struct {
	// IssueUID is the unique identifier of the issue.
	IssueUID *int
	Type     *api.ExternalApprovalType
}

// CreateExternalApprovalV2 creates an ExternalApproval.
func (s *Store) CreateExternalApprovalV2(ctx context.Context, create *ExternalApprovalMessage) (*ExternalApprovalMessage, error) {
	query := `
    INSERT INTO external_approval (
      issue_id,
      requester_id,
      approver_id,
      type,
      payload
    )
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, issue_id, approver_id, type, payload
  `

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var externalApproval ExternalApprovalMessage
	if err := tx.QueryRowContext(ctx, query,
		create.IssueUID,
		create.RequesterUID,
		create.ApproverUID,
		create.Type,
		create.Payload,
	).Scan(
		&externalApproval.ID,
		&externalApproval.IssueUID,
		&externalApproval.ApproverUID,
		&externalApproval.Type,
		&externalApproval.Payload,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	return &externalApproval, nil
}

// ListExternalApprovalV2 finds a list of ExternalApproval by find and whose RowStatus == NORMAL.
func (s *Store) ListExternalApprovalV2(ctx context.Context, find *ListExternalApprovalMessage) ([]*ExternalApprovalMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	externalApprovals, err := s.findExternalApprovalImplV2(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find external approval")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return externalApprovals, nil
}

// GetExternalApprovalByIssueIDV2 gets an ExternalApproval by IssueID.
func (s *Store) GetExternalApprovalByIssueIDV2(ctx context.Context, issueID int) (*ExternalApprovalMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	externalApprovals, err := s.findExternalApprovalImplV2(ctx, tx, &ListExternalApprovalMessage{IssueUID: &issueID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find external approval")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	if len(externalApprovals) == 0 {
		return nil, nil
	}
	if len(externalApprovals) > 1 {
		return nil, errors.Errorf("find %d external approvals for issue %d", len(externalApprovals), issueID)
	}
	return externalApprovals[0], nil
}

// DeleteExternalApprovalV2 updates an ExternalApproval.
func (s *Store) DeleteExternalApprovalV2(ctx context.Context, id int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM external_approval WHERE id = $1`, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	return nil
}

func (*Store) findExternalApprovalImplV2(ctx context.Context, tx *Tx, find *ListExternalApprovalMessage) ([]*ExternalApprovalMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.IssueUID; v != nil {
		where, args = append(where, fmt.Sprintf("issue_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		  id,
		  issue_id,
		  approver_id,
		  type,
		  payload
		FROM external_approval
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var externalApprovals []*ExternalApprovalMessage
	for rows.Next() {
		var externalApproval ExternalApprovalMessage
		if err := rows.Scan(
			&externalApproval.ID,
			&externalApproval.IssueUID,
			&externalApproval.ApproverUID,
			&externalApproval.Type,
			&externalApproval.Payload,
		); err != nil {
			return nil, err
		}
		externalApprovals = append(externalApprovals, &externalApproval)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return externalApprovals, nil
}
