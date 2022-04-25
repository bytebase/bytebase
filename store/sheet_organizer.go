package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// sheetOrganizerRaw is the store model for SheetOrganizer.
type sheetOrganizerRaw struct {
	ID int

	// Related fields
	SheetID     int
	PrincipalID int
	Starred     bool
	Pinned      bool
}

// toSheetOrganizer creates an instance of SheetOrganizer based on the sheetOrganizerRaw.
func (raw *sheetOrganizerRaw) toSheetOrganizer() *api.SheetOrganizer {
	return &api.SheetOrganizer{
		ID:          raw.ID,
		SheetID:     raw.SheetID,
		PrincipalID: raw.PrincipalID,
		Starred:     raw.Starred,
		Pinned:      raw.Pinned,
	}
}

// UpsertSheetOrganizer upserts a new SheetOrganizer.
func (s *Store) UpsertSheetOrganizer(ctx context.Context, upsert *api.SheetOrganizerUpsert) (*api.SheetOrganizer, error) {
	// TODO(steven): remove this release guard once ready to release.
	if s.db.mode != common.ReleaseModeDev {
		return nil, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	sheetOrganizerRaw, err := upsertSheetOrganizerImpl(ctx, tx.PTx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	sheetOrganizer := sheetOrganizerRaw.toSheetOrganizer()

	return sheetOrganizer, nil
}

// FindSheetOrganizer retrieves a SheetOrganizer.
func (s *Store) FindSheetOrganizer(ctx context.Context, find *api.SheetOrganizerFind) (*api.SheetOrganizer, error) {
	// TODO(steven): remove this release guard once ready to release.
	if s.db.mode != common.ReleaseModeDev {
		return nil, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	sheetOrganizerRawlist, err := findSheetOrganizerListImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(sheetOrganizerRawlist) == 0 {
		return nil, nil
	} else if len(sheetOrganizerRawlist) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d sheet organizer with filter %+v, expect 1. ", len(sheetOrganizerRawlist), find)}
	}

	sheetOrganizer := sheetOrganizerRawlist[0].toSheetOrganizer()

	return sheetOrganizer, nil
}

func upsertSheetOrganizerImpl(ctx context.Context, tx *sql.Tx, upsert *api.SheetOrganizerUpsert) (*sheetOrganizerRaw, error) {
	row, err := tx.QueryContext(ctx, `
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
	`,
		upsert.SheetID,
		upsert.PrincipalID,
		upsert.Starred,
		upsert.Pinned,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var sheetOrganizerRaw sheetOrganizerRaw
	if err := row.Scan(
		&sheetOrganizerRaw.ID,
		&sheetOrganizerRaw.SheetID,
		&sheetOrganizerRaw.PrincipalID,
		&sheetOrganizerRaw.Starred,
		&sheetOrganizerRaw.Pinned,
	); err != nil {
		return nil, FormatError(err)
	}

	return &sheetOrganizerRaw, nil
}

func findSheetOrganizerListImpl(ctx context.Context, tx *sql.Tx, find *api.SheetOrganizerFind) ([]*sheetOrganizerRaw, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, fmt.Sprintf("sheet_id = $%d", len(args)+1)), append(args, find.SheetID)
	where, args = append(where, fmt.Sprintf("principal_id = $%d", len(args)+1)), append(args, find.PrincipalID)

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
		return nil, FormatError(err)
	}
	defer rows.Close()

	var sheetOrganizerRawList []*sheetOrganizerRaw
	for rows.Next() {
		var sheetOrganizerRaw sheetOrganizerRaw
		if err := rows.Scan(
			&sheetOrganizerRaw.ID,
			&sheetOrganizerRaw.SheetID,
			&sheetOrganizerRaw.PrincipalID,
			&sheetOrganizerRaw.Starred,
			&sheetOrganizerRaw.Pinned,
		); err != nil {
			return nil, FormatError(err)
		}
		sheetOrganizerRawList = append(sheetOrganizerRawList, &sheetOrganizerRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return sheetOrganizerRawList, nil
}
