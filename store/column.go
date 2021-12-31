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
	_ api.ColumnService = (*ColumnService)(nil)
)

// ColumnService represents a service for managing column.
type ColumnService struct {
	l  *zap.Logger
	db *DB
}

// NewColumnService returns a new instance of ColumnService.
func NewColumnService(logger *zap.Logger, db *DB) *ColumnService {
	return &ColumnService{l: logger, db: db}
}

// CreateColumn creates a new column.
func (s *ColumnService) CreateColumn(ctx context.Context, create *api.ColumnCreate) (*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	column, err := s.createColumn(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return column, nil
}

// FindColumnList retrieves a list of columns based on find.
func (s *ColumnService) FindColumnList(ctx context.Context, find *api.ColumnFind) ([]*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findColumnList(ctx, tx, find)
	if err != nil {
		return []*api.Column{}, err
	}

	return list, nil
}

// FindColumn retrieves a single column based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ColumnService) FindColumn(ctx context.Context, find *api.ColumnFind) (*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findColumnList(ctx, tx, find)
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
func (s *ColumnService) PatchColumn(ctx context.Context, patch *api.ColumnPatch) (*api.Column, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	column, err := s.patchColumn(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return column, nil
}

// createColumn creates a new column.
func (s *ColumnService) createColumn(ctx context.Context, tx *Tx, create *api.ColumnCreate) (*api.Column, error) {
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
			`+"`default`,"+`
			`+"`nullable`,"+`
			`+"`type`,"+`
			character_set,
			collation,
			comment
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`+
		"RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, name, position, `default`, `nullable`, `type`, character_set, `collation`, comment"+`
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

func (s *ColumnService) findColumnList(ctx context.Context, tx *Tx, find *api.ColumnFind) (_ []*api.Column, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, "database_id = ?"), append(args, *v)
	}
	if v := find.TableID; v != nil {
		where, args = append(where, "table_id = ?"), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
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
			`+"`default`,"+`
			`+"`nullable`,"+`
			`+"`type`,"+`
			character_set,
			collation,
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Column, 0)
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

		list = append(list, &column)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchColumn updates a column by ID. Returns the new state of the column after update.
func (s *ColumnService) patchColumn(ctx context.Context, tx *Tx, patch *api.ColumnPatch) (*api.Column, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE col
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?`+
		"RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, name, position, `default`, `nullable`, `type`, character_set, `collation`, comment"+`
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
