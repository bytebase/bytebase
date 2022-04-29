package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// tableRaw is the store model for an Table.
// Fields have exactly the same meanings as Table.
type tableRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name          string
	Type          string
	Engine        string
	Collation     string
	RowCount      int64
	DataSize      int64
	IndexSize     int64
	DataFree      int64
	CreateOptions string
	Comment       string
}

// toTable creates an instance of Table based on the tableRaw.
// This is intended to be called when we need to compose an Table relationship.
func (raw *tableRaw) toTable() *api.Table {
	return &api.Table{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:          raw.Name,
		Type:          raw.Type,
		Engine:        raw.Engine,
		Collation:     raw.Collation,
		RowCount:      raw.RowCount,
		DataSize:      raw.DataSize,
		IndexSize:     raw.IndexSize,
		DataFree:      raw.DataFree,
		CreateOptions: raw.CreateOptions,
		Comment:       raw.Comment,
	}
}

// CreateTable creates an instance of Table
func (s *Store) CreateTable(ctx context.Context, create *api.TableCreate) (*api.Table, error) {
	tableRaw, err := s.createTableRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Table with TableCreate[%+v], error[%w]", create, err)
	}
	table, err := s.composeTable(ctx, tableRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Table with tableRaw[%+v], error[%w]", tableRaw, err)
	}
	return table, nil
}

// GetTable gets an instance of Table
func (s *Store) GetTable(ctx context.Context, find *api.TableFind) (*api.Table, error) {
	tableRaw, err := s.getTableRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Table with TableFind[%+v], error[%w]", find, err)
	}
	if tableRaw == nil {
		return nil, nil
	}
	table, err := s.composeTable(ctx, tableRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Table with tableRaw[%+v], error[%w]", tableRaw, err)
	}
	return table, nil
}

// FindTable finds a list of Table instances
func (s *Store) FindTable(ctx context.Context, find *api.TableFind) ([]*api.Table, error) {
	tableRawList, err := s.findTableRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Table list, error[%w]", err)
	}
	var tableList []*api.Table
	for _, raw := range tableRawList {
		table, err := s.composeTable(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Table with tableRaw[%+v], error[%w]", raw, err)
		}
		tableList = append(tableList, table)
	}
	return tableList, nil
}

// DeleteTable deletes an existing table by ID.
func (s *Store) DeleteTable(ctx context.Context, delete *api.TableDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteTableImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

//
// private functions
//

func (s *Store) composeTable(ctx context.Context, raw *tableRaw) (*api.Table, error) {
	table := raw.toTable()

	creator, err := s.GetPrincipalByID(ctx, table.CreatorID)
	if err != nil {
		return nil, err
	}
	table.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, table.UpdaterID)
	if err != nil {
		return nil, err
	}
	table.Updater = updater

	return table, nil
}

// createTableRaw creates a new table.
func (s *Store) createTableRaw(ctx context.Context, create *api.TableCreate) (*tableRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	table, err := s.createTableImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return table, nil
}

// findTableRaw retrieves a list of tables based on find.
func (s *Store) findTableRaw(ctx context.Context, find *api.TableFind) ([]*tableRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findTableImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getTableRaw retrieves a single table based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getTableRaw(ctx context.Context, find *api.TableFind) (*tableRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findTableImpl(ctx, tx.PTx, find)
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

// createTableImpl creates a new table.
func (s *Store) createTableImpl(ctx context.Context, tx *sql.Tx, create *api.TableCreate) (*tableRaw, error) {
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
	var tableRaw tableRaw
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

func (s *Store) findTableImpl(ctx context.Context, tx *sql.Tx, find *api.TableFind) ([]*tableRaw, error) {
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
	var tableRawList []*tableRaw
	for rows.Next() {
		var tableRaw tableRaw
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

// deleteTableImpl permanently deletes tables from a database.
func deleteTableImpl(ctx context.Context, tx *sql.Tx, delete *api.TableDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM tbl WHERE database_id = $1`, delete.DatabaseID); err != nil {
		return FormatError(err)
	}
	return nil
}
