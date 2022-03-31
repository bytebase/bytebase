package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.SheetService = (*SheetService)(nil)
)

// SheetService represents a service for managing sheet.
type SheetService struct {
	l  *zap.Logger
	db *DB
}

// NewSheetService returns a new sheet of SheetService.
func NewSheetService(logger *zap.Logger, db *DB) *SheetService {
	return &SheetService{l: logger, db: db}
}

// CreateSheet creates a new sheet.
func (s *SheetService) CreateSheet(ctx context.Context, create *api.SheetCreate) (*api.SheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	sheet, err := createSheet(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return sheet, nil
}

// PatchSheet updates an existing sheet by ID.
func (s *SheetService) PatchSheet(ctx context.Context, patch *api.SheetPatch) (*api.SheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	sheet, err := patchSheet(ctx, tx.PTx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return sheet, nil
}

// FindSheetList retrieves a list of sheet based on find.
func (s *SheetService) FindSheetList(ctx context.Context, find *api.SheetFind) ([]*api.SheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findSheetList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// FindSheet retrieves a single sheet based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *SheetService) FindSheet(ctx context.Context, find *api.SheetFind) (*api.SheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findSheetList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d sheet with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// DeleteSheet deletes an existing sheet by ID.
// Returns ENOTFOUND if sheet does not exist.
func (s *SheetService) DeleteSheet(ctx context.Context, delete *api.SheetDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteSheet(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createSheet creates a new sheet.
func createSheet(ctx context.Context, tx *sql.Tx, create *api.SheetCreate) (*api.SheetRaw, error) {
	// TODO(Steven): remove the default value when developing the service logic for sheet.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO sheet (
			creator_id,
			updater_id,
			project_id,
			database_id,
			name,
			statement,
			visibility,
			source,
			type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.DatabaseID,
		create.Name,
		create.Statement,
		create.Visibility,
		"BYTEBASE",
		"SQL",
		"{}",
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var sheetRaw api.SheetRaw
	databaseID := sql.NullInt32{}
	if err := row.Scan(
		&sheetRaw.ID,
		&sheetRaw.CreatorID,
		&sheetRaw.CreatedTs,
		&sheetRaw.UpdaterID,
		&sheetRaw.UpdatedTs,
		&sheetRaw.ProjectID,
		&databaseID,
		&sheetRaw.Name,
		&sheetRaw.Statement,
		&sheetRaw.Visibility,
		&sheetRaw.Source,
		&sheetRaw.Type,
		&sheetRaw.Payload,
	); err != nil {
		return nil, FormatError(err)
	}

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheetRaw.DatabaseID = &value
	}

	return &sheetRaw, nil
}

// patchSheet updates a sheet's name/statement/visibility.
func patchSheet(ctx context.Context, tx *sql.Tx, patch *api.SheetPatch) (*api.SheetRaw, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.Statement; v != nil {
		set, args = append(set, fmt.Sprintf("statement = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Visibility; v != nil {
		set, args = append(set, fmt.Sprintf("visibility = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload
	`, len(args)),
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var sheetRaw api.SheetRaw
		databaseID := sql.NullInt32{}
		if err := row.Scan(
			&sheetRaw.ID,
			&sheetRaw.CreatorID,
			&sheetRaw.CreatedTs,
			&sheetRaw.UpdaterID,
			&sheetRaw.UpdatedTs,
			&sheetRaw.ProjectID,
			&databaseID,
			&sheetRaw.Name,
			&sheetRaw.Statement,
			&sheetRaw.Visibility,
			&sheetRaw.Source,
			&sheetRaw.Type,
			&sheetRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheetRaw.DatabaseID = &value
		}

		return &sheetRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("sheet ID not found: %d", patch.ID)}
}

func findSheetList(ctx context.Context, tx *sql.Tx, find *api.SheetFind) ([]*api.SheetRaw, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	// Standard fields
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}

	// Related fields
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}

	// Domain fields
	if v := find.Visibility; v != nil {
		where, args = append(where, fmt.Sprintf("visibility = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Source; v != nil {
		where, args = append(where, fmt.Sprintf("source = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			project_id,
			database_id,
			name,
			statement,
			visibility,
			source,
			type,
			payload
		FROM sheet
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var sheetRawList []*api.SheetRaw
	for rows.Next() {
		var sheetRaw api.SheetRaw
		databaseID := sql.NullInt32{}
		if err := rows.Scan(
			&sheetRaw.ID,
			&sheetRaw.CreatorID,
			&sheetRaw.CreatedTs,
			&sheetRaw.UpdaterID,
			&sheetRaw.UpdatedTs,
			&sheetRaw.ProjectID,
			&databaseID,
			&sheetRaw.Name,
			&sheetRaw.Statement,
			&sheetRaw.Visibility,
			&sheetRaw.Source,
			&sheetRaw.Type,
			&sheetRaw.Payload,
		); err != nil {
			return nil, FormatError(err)
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheetRaw.DatabaseID = &value
		}

		sheetRawList = append(sheetRawList, &sheetRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return sheetRawList, nil
}

// deleteSheet permanently deletes a sheet by ID.
func deleteSheet(ctx context.Context, tx *sql.Tx, delete *api.SheetDelete) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM sheet WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
