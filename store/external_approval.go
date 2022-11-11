package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

type externalApprovalRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	IssueID     int
	RequesterID int
	ApproverID  int

	// Domain specific fields
	Type    api.ExternalApprovalType
	Payload string
}

func (raw *externalApprovalRaw) toExternalApproval() *api.ExternalApproval {
	return &api.ExternalApproval{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatedTs: raw.CreatedTs,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		IssueID:     raw.IssueID,
		RequesterID: raw.RequesterID,
		ApproverID:  raw.ApproverID,
		Type:        raw.Type,
		Payload:     raw.Payload,
	}
}

// CreateExternalApproval creates an ExternalApproval.
func (s *Store) CreateExternalApproval(ctx context.Context, create *api.ExternalApprovalCreate) (*api.ExternalApproval, error) {
	externalApprovalRaw, err := s.createExternalApprovalRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ExternalApproval with ExternalApprovalCreate[%+v]", create)
	}
	externalApproval, err := s.composeExternalApproval(ctx, externalApprovalRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose ExternalApproval with externalApprovalRaw[%+v]", externalApprovalRaw)
	}
	return externalApproval, nil
}

// FindExternalApproval finds a list of ExternalApproval by find and whose RowStatus == NORMAL.
func (s *Store) FindExternalApproval(ctx context.Context, find *api.ExternalApprovalFind) ([]*api.ExternalApproval, error) {
	rawList, err := s.findExternalApprovalRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find ExternalApproval with ExternalApprovalFind[%+v]", find)
	}
	var externalApprovalList []*api.ExternalApproval
	for _, raw := range rawList {
		externalApproval, err := s.composeExternalApproval(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose ExternalApproval with externalApprovalRaw[%+v]", raw)
		}
		externalApprovalList = append(externalApprovalList, externalApproval)
	}
	return externalApprovalList, nil
}

// GetExternalApprovalByIssueID gets an ExternalApproval by IssueID.
func (s *Store) GetExternalApprovalByIssueID(ctx context.Context, issueID int) (*api.ExternalApproval, error) {
	externalApprovalRaw, err := s.getExternalApprovalRawByIssueID(ctx, issueID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ApprovalInstance by issueID %v", issueID)
	}
	if externalApprovalRaw == nil {
		return nil, nil
	}
	externalApproval, err := s.composeExternalApproval(ctx, externalApprovalRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose ExternalApproval with externalApprovalRaw[%+v]", externalApprovalRaw)
	}
	return externalApproval, nil
}

// PatchExternalApproval patches an ExternalApproval.
func (s *Store) PatchExternalApproval(ctx context.Context, patch *api.ExternalApprovalPatch) (*api.ExternalApproval, error) {
	externalApprovalRaw, err := s.patchExternalApprovalRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch ExternalApproval with ExternalApprovalPatch[%+v]", patch)
	}
	externalApproval, err := s.composeExternalApproval(ctx, externalApprovalRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose ExternalApproval with externalApprovalRaw[%+v]", externalApprovalRaw)
	}
	return externalApproval, nil
}

//
// private functions
//

func (s *Store) composeExternalApproval(ctx context.Context, raw *externalApprovalRaw) (*api.ExternalApproval, error) {
	externalApproval := raw.toExternalApproval()

	requester, err := s.GetPrincipalByID(ctx, externalApproval.RequesterID)
	if err != nil {
		return nil, err
	}
	externalApproval.Requester = requester

	approver, err := s.GetPrincipalByID(ctx, externalApproval.ApproverID)
	if err != nil {
		return nil, err
	}
	externalApproval.Approver = approver

	return externalApproval, nil
}

func (s *Store) createExternalApprovalRaw(ctx context.Context, create *api.ExternalApprovalCreate) (*externalApprovalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	externalApprovalRaw, err := s.createExternalApprovalImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return externalApprovalRaw, nil
}

func (s *Store) findExternalApprovalRaw(ctx context.Context, find *api.ExternalApprovalFind) ([]*externalApprovalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rawList, err := s.findExternalApprovalImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return rawList, nil
}

func (s *Store) getExternalApprovalRawByIssueID(ctx context.Context, issueID int) (*externalApprovalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()
	rawList, err := s.findExternalApprovalImpl(ctx, tx, &api.ExternalApprovalFind{IssueID: &issueID})
	if err != nil {
		return nil, err
	}

	if len(rawList) == 0 {
		return nil, nil
	} else if len(rawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d externalApprovals with issueID %d, expect 1", len(rawList), issueID)}
	}
	return rawList[0], nil
}

func (s *Store) patchExternalApprovalRaw(ctx context.Context, patch *api.ExternalApprovalPatch) (*externalApprovalRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()
	raw, err := s.patchExternalApprovalImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return raw, nil
}

func (*Store) createExternalApprovalImpl(ctx context.Context, tx *Tx, create *api.ExternalApprovalCreate) (*externalApprovalRaw, error) {
	query := `
    INSERT INTO external_approval (
      issue_id,
      requester_id,
      approver_id,
      type,
      payload
    )
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, row_status, created_ts, updated_ts, issue_id, requester_id, approver_id, type, payload
  `
	var externalApprovalRaw externalApprovalRaw
	if err := tx.QueryRowContext(ctx, query,
		create.IssueID,
		create.RequesterID,
		create.ApproverID,
		create.Type,
		create.Payload,
	).Scan(
		&externalApprovalRaw.ID,
		&externalApprovalRaw.RowStatus,
		&externalApprovalRaw.CreatedTs,
		&externalApprovalRaw.UpdatedTs,
		&externalApprovalRaw.IssueID,
		&externalApprovalRaw.RequesterID,
		&externalApprovalRaw.ApproverID,
		&externalApprovalRaw.Type,
		&externalApprovalRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}
	return &externalApprovalRaw, nil
}

func (*Store) findExternalApprovalImpl(ctx context.Context, tx *Tx, find *api.ExternalApprovalFind) ([]*externalApprovalRaw, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.Normal)
	if v := find.IssueID; v != nil {
		where, args = append(where, fmt.Sprintf("issue_id = $%d", len(args)+1)), append(args, *v)
	}
	rows, err := tx.QueryContext(ctx, `
    SELECT
      id,
      row_status,
      created_ts,
      updated_ts,
      issue_id,
      requester_id,
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
	var rawList []*externalApprovalRaw
	for rows.Next() {
		var raw externalApprovalRaw
		if err := rows.Scan(
			&raw.ID,
			&raw.RowStatus,
			&raw.CreatedTs,
			&raw.UpdatedTs,
			&raw.IssueID,
			&raw.RequesterID,
			&raw.ApproverID,
			&raw.Type,
			&raw.Payload,
		); err != nil {
			return nil, err
		}
		rawList = append(rawList, &raw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return rawList, nil
}

func (*Store) patchExternalApprovalImpl(ctx context.Context, tx *Tx, patch *api.ExternalApprovalPatch) (*externalApprovalRaw, error) {
	var raw externalApprovalRaw
	query := `
    UPDATE external_approval
    SET row_status = $1
    WHERE id = $2
    RETURNING id, row_status, created_ts, updated_ts, issue_id, requester_id, approver_id, type, payload
  `
	if err := tx.QueryRowContext(ctx, query, patch.RowStatus, patch.ID).Scan(
		&raw.ID,
		&raw.RowStatus,
		&raw.CreatedTs,
		&raw.UpdatedTs,
		&raw.IssueID,
		&raw.RequesterID,
		&raw.ApproverID,
		&raw.Type,
		&raw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}
	return &raw, nil
}
