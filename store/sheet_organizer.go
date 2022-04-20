package store

import (
	"context"
	"database/sql"

	"github.com/bytebase/bytebase/api"
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	sheetOrganizerRaw, err := upsertSheetOrganizer(ctx, tx.PTx, upsert)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	sheetOrganizer := sheetOrganizerRaw.toSheetOrganizer()

	return sheetOrganizer, nil
}

func upsertSheetOrganizer(ctx context.Context, tx *sql.Tx, upsert *api.SheetOrganizerUpsert) (*sheetOrganizerRaw, error) {
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
