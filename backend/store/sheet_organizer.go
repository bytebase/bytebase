package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// SheetOrganizerMessage is the store message for sheet organizer.
type SheetOrganizerMessage struct {
	UID int

	// Related fields
	SheetUID     int
	PrincipalUID int
	Starred      bool
	Pinned       bool
}

// FindSheetOrganizerMessage is the store message to find sheet organizer.
type FindSheetOrganizerMessage struct {
	// Related fields
	SheetUID     int
	PrincipalUID int
}

// UpsertSheetOrganizerV2 upserts a new SheetOrganizerMessage.
func (s *Store) UpsertSheetOrganizerV2(ctx context.Context, organizer *SheetOrganizerMessage) (*SheetOrganizerMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
	  INSERT INTO sheet_organizer (
			sheet_id,
			principal_id,
			starred,
			pinned
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(sheet_id, principal_id) DO UPDATE SET
			starred = EXCLUDED.starred,
			pinned = EXCLUDED.pinned
		RETURNING id, sheet_id, principal_id, starred, pinned
	`
	var sheetOrganizer SheetOrganizerMessage
	if err := tx.QueryRowContext(ctx, query,
		organizer.SheetUID,
		organizer.PrincipalUID,
		organizer.Starred,
		organizer.Pinned,
	).Scan(
		&sheetOrganizer.UID,
		&sheetOrganizer.SheetUID,
		&sheetOrganizer.PrincipalUID,
		&sheetOrganizer.Starred,
		&sheetOrganizer.Pinned,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &sheetOrganizer, nil
}

// FindSheetOrganizerV2 retrieves a SheetOrganizerMessage.
func (s *Store) FindSheetOrganizerV2(ctx context.Context, find *FindSheetOrganizerMessage) (*SheetOrganizerMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("sheet_id = $%d", len(args)+1)), append(args, find.SheetUID)
	where, args = append(where, fmt.Sprintf("principal_id = $%d", len(args)+1)), append(args, find.PrincipalUID)

	rows, err := tx.QueryContext(ctx, `
	SELECT
		id,
		sheet_id,
		principal_id,
		starred,
		pinned
	FROM sheet_organizer
	WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheetOrganizerlist []*SheetOrganizerMessage
	for rows.Next() {
		var sheetOrganizer SheetOrganizerMessage
		if err := rows.Scan(
			&sheetOrganizer.UID,
			&sheetOrganizer.SheetUID,
			&sheetOrganizer.PrincipalUID,
			&sheetOrganizer.Starred,
			&sheetOrganizer.Pinned,
		); err != nil {
			return nil, err
		}
		sheetOrganizerlist = append(sheetOrganizerlist, &sheetOrganizer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if len(sheetOrganizerlist) == 0 {
		return nil, nil
	} else if len(sheetOrganizerlist) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d sheet organizer with filter %+v, expect 1. ", len(sheetOrganizerlist), find)}
	}

	return sheetOrganizerlist[0], nil
}
