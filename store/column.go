package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// CreateColumn creates a new column.
func (s *Store) CreateColumn(ctx context.Context, create *api.ColumnCreate) (*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	column, err := s.createColumnImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return column, nil
}

// FindColumn retrieves a list of columns based on find.
func (s *Store) FindColumn(ctx context.Context, find *api.ColumnFind) ([]*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findColumnImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// GetColumn retrieves a single column based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) GetColumn(ctx context.Context, find *api.ColumnFind) (*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findColumnImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d columns with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchColumn updates an existing column by ID.
// Returns ENOTFOUND if column does not exist.
func (s *Store) PatchColumn(ctx context.Context, patch *api.ColumnPatch) (*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	column, err := s.patchColumn(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return column, nil
}

// createColumnImpl creates a new column.
func (s *Store) createColumnImpl(ctx context.Context, tx *sql.Tx, create *api.ColumnCreate) (*api.Column, error) {
	defaultStr := sql.NullString{}
	if create.Default != nil {
		defaultStr = sql.NullString{
			String: *create.Default,
			Valid:  true,
		}
	}

	// Insert row into column.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO col (
			creator_id,
			updater_id,
			database_id,
			table_id,
			name,
			position,
			"default",
			nullable,
			type,
			character_set,
			"collation",
			comment
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, name, position, "default", nullable, type, character_set, "collation", comment
	`,
		create.CreatorID,
		create.CreatorID,
		create.DatabaseID,
		create.TableID,
		create.Name,
		create.Position,
		defaultStr,
		create.Nullable,
		create.Type,
		create.CharacterSet,
		create.Collation,
		create.Comment,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var column api.Column
	if err := row.Scan(
		&column.ID,
		&column.CreatorID,
		&column.CreatedTs,
		&column.UpdaterID,
		&column.UpdatedTs,
		&column.DatabaseID,
		&column.TableID,
		&column.Name,
		&column.Position,
		&defaultStr,
		&column.Nullable,
		&column.Type,
		&column.CharacterSet,
		&column.Collation,
		&column.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	if defaultStr.Valid {
		column.Default = &defaultStr.String
	}

	return &column, nil
}

func (s *Store) findColumnImpl(ctx context.Context, tx *sql.Tx, find *api.ColumnFind) ([]*api.Column, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.TableID; v != nil {
		where, args = append(where, fmt.Sprintf("table_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
			table_id,
			name,
			position,
			"default",
			nullable,
			type,
			character_set,
			"collation",
			comment
		FROM col
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY database_id, table_id, position ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into columnList.
	var columnList []*api.Column
	for rows.Next() {
		var column api.Column
		defaultStr := sql.NullString{}
		if err := rows.Scan(
			&column.ID,
			&column.CreatorID,
			&column.CreatedTs,
			&column.UpdaterID,
			&column.UpdatedTs,
			&column.DatabaseID,
			&column.TableID,
			&column.Name,
			&column.Position,
			&defaultStr,
			&column.Nullable,
			&column.Type,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		if defaultStr.Valid {
			column.Default = &defaultStr.String
		}

		columnList = append(columnList, &column)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return columnList, nil
}

// patchColumn updates a column by ID. Returns the new state of the column after update.
func (s *Store) patchColumn(ctx context.Context, tx *sql.Tx, patch *api.ColumnPatch) (*api.Column, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE col
		SET `+strings.Join(set, ", ")+`
		WHERE id = $2
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, name, position, "default", nullable, type, character_set, "collation", comment
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var column api.Column
		defaultStr := sql.NullString{}
		if err := row.Scan(
			&column.ID,
			&column.CreatorID,
			&column.CreatedTs,
			&column.UpdaterID,
			&column.UpdatedTs,
			&column.DatabaseID,
			&column.TableID,
			&column.Name,
			&column.Position,
			&defaultStr,
			&column.Nullable,
			&column.Type,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		if defaultStr.Valid {
			column.Default = &defaultStr.String
		}

		return &column, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("column ID not found: %d", patch.ID)}
}
