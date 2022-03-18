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
func (s *TableService) CreateTable(ctx context.Context, create *api.TableCreate) (*api.TableRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	table, err := s.createTable(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return table, nil
}

// FindTableList retrieves a list of tables based on find.
func (s *TableService) FindTableList(ctx context.Context, find *api.TableFind) ([]*api.TableRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findTableList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// FindTable retrieves a single table based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *TableService) FindTable(ctx context.Context, find *api.TableFind) (*api.TableRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findTableList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d tables with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// DeleteTable deletes an existing table by ID.
func (s *TableService) DeleteTable(ctx context.Context, delete *api.TableDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteTable(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createTable creates a new table.
func (s *TableService) createTable(ctx context.Context, tx *sql.Tx, create *api.TableCreate) (*api.TableRaw, error) {
	// Insert row into table.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO tbl (
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
			name,
			type,
			engine,
			"collation",
			row_count,
			data_size,
			index_size,
			data_free,
			create_options,
			comment
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, type, engine, "collation", row_count, data_size, index_size, data_free, create_options, comment
	`,
		create.CreatorID,
		create.CreatedTs,
		create.CreatorID,
		create.UpdatedTs,
		create.DatabaseID,
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
	var tableRaw api.TableRaw
	if err := row.Scan(
		&tableRaw.ID,
		&tableRaw.CreatorID,
		&tableRaw.CreatedTs,
		&tableRaw.UpdaterID,
		&tableRaw.UpdatedTs,
		&tableRaw.DatabaseID,
		&tableRaw.Name,
		&tableRaw.Type,
		&tableRaw.Engine,
		&tableRaw.Collation,
		&tableRaw.RowCount,
		&tableRaw.DataSize,
		&tableRaw.IndexSize,
		&tableRaw.DataFree,
		&tableRaw.CreateOptions,
		&tableRaw.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	return &tableRaw, nil
}

func (s *TableService) findTableList(ctx context.Context, tx *sql.Tx, find *api.TableFind) ([]*api.TableRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
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
			name,
			type,
			engine,
			"collation",
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

	// Iterate over result set and deserialize rows into tableRawList.
	var tableRawList []*api.TableRaw
	for rows.Next() {
		var tableRaw api.TableRaw
		if err := rows.Scan(
			&tableRaw.ID,
			&tableRaw.CreatorID,
			&tableRaw.CreatedTs,
			&tableRaw.UpdaterID,
			&tableRaw.UpdatedTs,
			&tableRaw.DatabaseID,
			&tableRaw.Name,
			&tableRaw.Type,
			&tableRaw.Engine,
			&tableRaw.Collation,
			&tableRaw.RowCount,
			&tableRaw.DataSize,
			&tableRaw.IndexSize,
			&tableRaw.DataFree,
			&tableRaw.CreateOptions,
			&tableRaw.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		tableRawList = append(tableRawList, &tableRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return tableRawList, nil
}

// deleteTable permanently deletes tables from a database.
func deleteTable(ctx context.Context, tx *sql.Tx, delete *api.TableDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM tbl WHERE database_id = $1`, delete.DatabaseID); err != nil {
		return FormatError(err)
	}
	return nil
}
