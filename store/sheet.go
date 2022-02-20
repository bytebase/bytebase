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
func (s *SheetService) CreateSheet(ctx context.Context, create *api.SheetCreate) (*api.Sheet, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	sheet, err := createSheet(ctx, tx.Tx, create)
	if err != nil {
		return nil, err
	}
	if _, err := pgCreateSheet(ctx, tx.PTx, create); err != nil {
		return nil, err
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return sheet, nil
}

// PatchSheet updates an existing sheet by ID.
func (s *SheetService) PatchSheet(ctx context.Context, patch *api.SheetPatch) (*api.Sheet, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	sheet, err := patchSheet(ctx, tx.Tx, patch)
	if err != nil {
		return nil, err
	}
	if _, err := pgPatchSheet(ctx, tx.PTx, patch); err != nil {
		return nil, err
	}

	if err := tx.Tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return sheet, nil
}

// FindSheetList retrieves a list of sheet based on find.
func (s *SheetService) FindSheetList(ctx context.Context, find *api.SheetFind) ([]*api.Sheet, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := findSheetList(ctx, tx, find)
	if err != nil {
		return []*api.Sheet{}, err
	}

	return list, nil
}

// FindSheet retrieves a single sheet based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *SheetService) FindSheet(ctx context.Context, find *api.SheetFind) (*api.Sheet, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	list, err := findSheetList(ctx, tx, find)
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
	defer tx.Tx.Rollback()
	defer tx.PTx.Rollback()

	if err := deleteSheet(ctx, tx.Tx, delete); err != nil {
		return FormatError(err)
	}
	if err := pgDeleteSheet(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.Tx.Commit(); err != nil {
		return FormatError(err)
	}
	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createSheet creates a new sheet.
func createSheet(ctx context.Context, tx *sql.Tx, create *api.SheetCreate) (*api.Sheet, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO sheet (
			creator_id,
			updater_id,
			instance_id,
			database_id,
			name,
			statement,
			visibility
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, statement, visibility
	`,
		create.CreatorID,
		create.CreatorID,
		create.InstanceID,
		create.DatabaseID,
		create.Name,
		create.Statement,
		create.Visibility,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var sheet api.Sheet
	databaseID := sql.NullInt32{}
	if err := row.Scan(
		&sheet.ID,
		&sheet.CreatorID,
		&sheet.CreatedTs,
		&sheet.UpdaterID,
		&sheet.UpdatedTs,
		&sheet.InstanceID,
		&databaseID,
		&sheet.Name,
		&sheet.Statement,
		&sheet.Visibility,
	); err != nil {
		return nil, FormatError(err)
	}

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheet.DatabaseID = &value
	}

	return &sheet, nil
}

// pgCreateSheet creates a new sheet.
func pgCreateSheet(ctx context.Context, tx *sql.Tx, create *api.SheetCreate) (*api.Sheet, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO sheet (
			creator_id,
			updater_id,
			instance_id,
			database_id,
			name,
			statement,
			visibility
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, statement, visibility
	`,
		create.CreatorID,
		create.CreatorID,
		create.InstanceID,
		create.DatabaseID,
		create.Name,
		create.Statement,
		create.Visibility,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var sheet api.Sheet
	databaseID := sql.NullInt32{}
	if err := row.Scan(
		&sheet.ID,
		&sheet.CreatorID,
		&sheet.CreatedTs,
		&sheet.UpdaterID,
		&sheet.UpdatedTs,
		&sheet.InstanceID,
		&databaseID,
		&sheet.Name,
		&sheet.Statement,
		&sheet.Visibility,
	); err != nil {
		return nil, FormatError(err)
	}

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheet.DatabaseID = &value
	}

	return &sheet, nil
}

// patchSheet creates a new sheet.
func patchSheet(ctx context.Context, tx *sql.Tx, patch *api.SheetPatch) (*api.Sheet, error) {
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, api.RowStatus(*v))
	}
	if v := patch.Statement; v != nil {
		set, args = append(set, "statement = ?"), append(args, *v)
	}
	if v := patch.Visibility; v != nil {
		set, args = append(set, "visibility = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	row, err := tx.QueryContext(ctx, `
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, statement, visibility
	`,
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var sheet api.Sheet
		databaseID := sql.NullInt32{}
		if err := row.Scan(
			&sheet.ID,
			&sheet.CreatorID,
			&sheet.CreatedTs,
			&sheet.UpdaterID,
			&sheet.UpdatedTs,
			&sheet.InstanceID,
			&databaseID,
			&sheet.Name,
			&sheet.Statement,
			&sheet.Visibility,
		); err != nil {
			return nil, FormatError(err)
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheet.DatabaseID = &value
		}

		return &sheet, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("sheet ID not found: %d", patch.ID)}
}

// pgPatchSheet creates a new sheet.
func pgPatchSheet(ctx context.Context, tx *sql.Tx, patch *api.SheetPatch) (*api.Sheet, error) {
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

	args = append(args, patch.ID)

	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, instance_id, database_id, name, statement, visibility
	`, len(args)),
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var sheet api.Sheet
		databaseID := sql.NullInt32{}
		if err := row.Scan(
			&sheet.ID,
			&sheet.CreatorID,
			&sheet.CreatedTs,
			&sheet.UpdaterID,
			&sheet.UpdatedTs,
			&sheet.InstanceID,
			&databaseID,
			&sheet.Name,
			&sheet.Statement,
			&sheet.Visibility,
		); err != nil {
			return nil, FormatError(err)
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheet.DatabaseID = &value
		}

		return &sheet, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("sheet ID not found: %d", patch.ID)}
}

func findSheetList(ctx context.Context, tx *Tx, find *api.SheetFind) (_ []*api.Sheet, err error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	// Standard fields
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = ?"), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "creator_id = ?"), append(args, *v)
	}

	// Related fields
	if v := find.InstanceID; v != nil {
		where, args = append(where, "instance_id = ?"), append(args, *v)
		if find.InstanceOnly {
			where = append(where, "database_id is NULL")
		}
	}
	if find.InstanceID == nil || !find.InstanceOnly {
		if v := find.DatabaseID; v != nil {
			where, args = append(where, "database_id = ?"), append(args, *v)
		}
	}

	// Domain fields
	if v := find.Visibility; v != nil {
		where, args = append(where, "visibility = ?"), append(args, *v)
	}

	rows, err := tx.Tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			instance_id,
			database_id,
			name,
			statement,
			visibility
		FROM sheet
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	list := make([]*api.Sheet, 0)
	for rows.Next() {
		var sheet api.Sheet
		databaseID := sql.NullInt32{}
		if err := rows.Scan(
			&sheet.ID,
			&sheet.CreatorID,
			&sheet.CreatedTs,
			&sheet.UpdaterID,
			&sheet.UpdatedTs,
			&sheet.InstanceID,
			&databaseID,
			&sheet.Name,
			&sheet.Statement,
			&sheet.Visibility,
		); err != nil {
			return nil, FormatError(err)
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheet.DatabaseID = &value
		}

		list = append(list, &sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// deleteSheet permanently deletes a sheet by ID.
func deleteSheet(ctx context.Context, tx *sql.Tx, delete *api.SheetDelete) error {
	result, err := tx.ExecContext(ctx, `DELETE FROM sheet WHERE id = ?`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("sheet ID not found: %d", delete.ID)}
	}

	return nil
}

// pgDeleteSheet permanently deletes a sheet by ID.
func pgDeleteSheet(ctx context.Context, tx *sql.Tx, delete *api.SheetDelete) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM sheet WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
