package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.TableService = (*TableService)(nil)
)

// TableService represents a service for managing table.
type TableService struct {
	l  *zap.Logger
	db *DB
}

// NewTableService returns a new instance of TableService.
func NewTableService(logger *zap.Logger, db *DB) *TableService {
	return &TableService{l: logger, db: db}
}

// CreateTable creates a new table.
func (s *TableService) CreateTable(ctx context.Context, create *api.TableCreate) (*api.Table, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	table, err := s.createTable(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return table, nil
}

// FindTableList retrieves a list of tables based on find.
func (s *TableService) FindTableList(ctx context.Context, find *api.TableFind) ([]*api.Table, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTableList(ctx, tx, find)
	if err != nil {
		return []*api.Table{}, err
	}

	return list, nil
}

// FindTable retrieves a single table based on find.
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *TableService) FindTable(ctx context.Context, find *api.TableFind) (*api.Table, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findTableList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &common.Error{Code: common.NotFound, Message: fmt.Sprintf("table not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Message: fmt.Sprintf("found %d tables with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchTable updates an existing table by ID.
// Returns ENOTFOUND if table does not exist.
func (s *TableService) PatchTable(ctx context.Context, patch *api.TablePatch) (*api.Table, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	table, err := s.patchTable(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return table, nil
}

// DeleteTable deletes an existing table by ID.
// Returns ENOTFOUND if table does not exist.
func (s *TableService) DeleteTable(ctx context.Context, delete *api.TableDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteTable(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createTable creates a new table.
func (s *TableService) createTable(ctx context.Context, tx *Tx, create *api.TableCreate) (*api.Table, error) {
	// Insert row into table.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO tbl (
			creator_id,
			updater_id,
			database_id,
			name,
			`+"`type`,"+`
			engine,
			collation,
			row_count,
			data_size,
			index_size,
			data_free,
			create_options,
			comment
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, `+"`type`, engine, collation, row_count, data_size, index_size, data_free, create_options, comment"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.DatabaseId,
		create.Name,
		create.Type,
		create.Engine,
		create.Collation,
		create.RowCount,
		create.DataSize,
		create.IndexSize,
		create.DataFree,
		create.CreateOptions,
		create.Comment,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var table api.Table
	if err := row.Scan(
		&table.ID,
		&table.CreatorId,
		&table.CreatedTs,
		&table.UpdaterId,
		&table.UpdatedTs,
		&table.DatabaseId,
		&table.Name,
		&table.Type,
		&table.Engine,
		&table.Collation,
		&table.RowCount,
		&table.DataSize,
		&table.IndexSize,
		&table.DataFree,
		&table.CreateOptions,
		&table.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	return &table, nil
}

func (s *TableService) findTableList(ctx context.Context, tx *Tx, find *api.TableFind) (_ []*api.Table, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.DatabaseId; v != nil {
		where, args = append(where, "database_id = ?"), append(args, *v)
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
		    name,
			`+"`type`,"+`
			engine,
			collation,
			row_count,
			data_size,
			index_size,
			data_free,
			create_options,
			comment
		FROM tbl
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Table, 0)
	for rows.Next() {
		var table api.Table
		if err := rows.Scan(
			&table.ID,
			&table.CreatorId,
			&table.CreatedTs,
			&table.UpdaterId,
			&table.UpdatedTs,
			&table.DatabaseId,
			&table.Name,
			&table.Type,
			&table.Engine,
			&table.Collation,
			&table.RowCount,
			&table.DataSize,
			&table.IndexSize,
			&table.DataFree,
			&table.CreateOptions,
			&table.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &table)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchTable updates a table by ID. Returns the new state of the table after update.
func (s *TableService) patchTable(ctx context.Context, tx *Tx, patch *api.TablePatch) (*api.Table, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE tbl
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, `+"`type`, engine, collation, row_count, data_size, index_size, data_free, create_options, comment"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var table api.Table
		if err := row.Scan(
			&table.ID,
			&table.CreatorId,
			&table.CreatedTs,
			&table.UpdaterId,
			&table.UpdatedTs,
			&table.DatabaseId,
			&table.Name,
			&table.Type,
			&table.Engine,
			&table.Collation,
			&table.RowCount,
			&table.DataSize,
			&table.IndexSize,
			&table.DataFree,
			&table.CreateOptions,
			&table.Comment,
		); err != nil {
			return nil, FormatError(err)
		}
		return &table, nil
	}

	return nil, &common.Error{Code: common.NotFound, Message: fmt.Sprintf("table ID not found: %d", patch.ID)}
}

// deleteTable permanently deletes tables from a database.
func deleteTable(ctx context.Context, tx *Tx, delete *api.TableDelete) error {
	// Remove row from database.
	_, err := tx.ExecContext(ctx, `DELETE FROM tbl WHERE database_id = ?`, delete.DatabaseId)
	if err != nil {
		return FormatError(err)
	}

	return nil
}
