package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
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

// GetTable gets an instance of Table.
func (s *Store) GetTable(ctx context.Context, find *api.TableFind) (*api.Table, error) {
	tableRaw, err := s.getTableRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Table with TableFind[%+v], error: %w", find, err)
	}
	if tableRaw == nil {
		return nil, nil
	}
	table, err := s.composeTable(ctx, tableRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Table with tableRaw[%+v], error: %w", tableRaw, err)
	}
	return table, nil
}

// FindTable finds a list of Table instances.
func (s *Store) FindTable(ctx context.Context, find *api.TableFind) ([]*api.Table, error) {
	tableRawList, err := s.findTableRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Table list, error: %w", err)
	}
	var tableList []*api.Table
	for _, raw := range tableRawList {
		table, err := s.composeTable(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Table with tableRaw[%+v], error: %w", raw, err)
		}
		tableList = append(tableList, table)
	}
	return tableList, nil
}

// SetTableList sets the tables for a database.
func (s *Store) SetTableList(ctx context.Context, schema *db.Schema, databaseID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	oldTableRawList, err := s.findTableImpl(ctx, tx.PTx, &api.TableFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return FormatError(err)
	}
	creates, patches, deletes := generateTableActions(oldTableRawList, schema.TableList, databaseID)
	for _, d := range deletes {
		if err := s.deleteTableImpl(ctx, tx.PTx, d); err != nil {
			return err
		}
	}
	for _, p := range patches {
		if _, err := s.patchTableImpl(ctx, tx.PTx, p); err != nil {
			return err
		}
	}
	for _, c := range creates {
		if _, err := s.createTableImpl(ctx, tx.PTx, c); err != nil {
			return err
		}
	}

	tableRawList, err := s.findTableImpl(ctx, tx.PTx, &api.TableFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return err
	}
	tableIDMap := make(map[string]int)
	for _, table := range tableRawList {
		tableIDMap[table.Name] = table.ID
	}

	for _, table := range schema.TableList {
		tableID, ok := tableIDMap[table.Name]
		if !ok {
			log.Error(fmt.Sprintf("table ID cannot be found for database %v table %s", databaseID, table.Name))
		}

		columnList, err := s.findColumnImpl(ctx, tx.PTx, &api.ColumnFind{
			TableID: &tableID,
		})
		if err != nil {
			return err
		}
		deletes, creates := generateColumnActions(columnList, table.ColumnList, databaseID, tableID)
		for _, d := range deletes {
			if err := s.deleteColumnImpl(ctx, tx.PTx, d); err != nil {
				return err
			}
		}
		for _, c := range creates {
			if _, err := s.createColumnImpl(ctx, tx.PTx, c); err != nil {
				return err
			}
		}

		indexList, err := s.findIndexImpl(ctx, tx.PTx, &api.IndexFind{
			TableID: &tableID,
		})
		if err != nil {
			return err
		}
		idxDeletes, idxCreates := generateIndexActions(indexList, table.IndexList, databaseID, tableID)
		for _, d := range idxDeletes {
			if err := s.deleteIndexImpl(ctx, tx.PTx, d); err != nil {
				return err
			}
		}
		for _, c := range idxCreates {
			if _, err := s.createIndexImpl(ctx, tx.PTx, c); err != nil {
				return err
			}
		}
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

func generateTableActions(oldTableRawList []*tableRaw, tableList []db.Table, databaseID int) ([]*api.TableCreate, []*api.TablePatch, []*api.TableDelete) {
	oldTableMap := make(map[string]*tableRaw)
	for _, t := range oldTableRawList {
		oldTableMap[t.Name] = t
	}
	newTableMap := make(map[string]db.Table)
	for _, t := range tableList {
		newTableMap[t.Name] = t
	}

	var creates []*api.TableCreate
	var patches []*api.TablePatch
	var deletes []*api.TableDelete
	for _, oldValue := range oldTableRawList {
		k := oldValue.Name
		newValue, ok := newTableMap[k]
		if !ok {
			deletes = append(deletes, &api.TableDelete{ID: oldValue.ID})
		} else if ok &&
			(oldValue.Type != newValue.Type ||
				oldValue.Engine != newValue.Engine ||
				oldValue.Collation != newValue.Collation ||
				oldValue.RowCount != newValue.RowCount ||
				oldValue.DataSize != newValue.DataSize ||
				oldValue.IndexSize != newValue.IndexSize ||
				oldValue.DataFree != newValue.DataFree ||
				oldValue.CreateOptions != newValue.CreateOptions ||
				oldValue.Comment != newValue.Comment) {
			patches = append(patches,
				&api.TablePatch{
					ID:            oldValue.ID,
					UpdaterID:     api.SystemBotID,
					Type:          newValue.Type,
					Engine:        newValue.Engine,
					Collation:     newValue.Collation,
					RowCount:      newValue.RowCount,
					DataSize:      newValue.DataSize,
					IndexSize:     newValue.IndexSize,
					DataFree:      newValue.DataFree,
					CreateOptions: newValue.CreateOptions,
					Comment:       newValue.Comment,
				},
			)
		}
	}
	for _, newValue := range tableList {
		k := newValue.Name
		if _, ok := oldTableMap[k]; !ok {
			creates = append(creates, &api.TableCreate{
				CreatorID:     api.SystemBotID,
				CreatedTs:     newValue.CreatedTs,
				UpdatedTs:     newValue.UpdatedTs,
				DatabaseID:    databaseID,
				Name:          newValue.Name,
				Type:          newValue.Type,
				Engine:        newValue.Engine,
				Collation:     newValue.Collation,
				RowCount:      newValue.RowCount,
				DataSize:      newValue.DataSize,
				IndexSize:     newValue.IndexSize,
				DataFree:      newValue.DataFree,
				CreateOptions: newValue.CreateOptions,
				Comment:       newValue.Comment,
			})
		}
	}
	return creates, patches, deletes
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

	database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: &table.DatabaseID})
	if err != nil {
		return nil, err
	}
	table.Database = database

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
func (*Store) createTableImpl(ctx context.Context, tx *sql.Tx, create *api.TableCreate) (*tableRaw, error) {
	// Insert row into table.
	query := `
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
	`
	var tableRaw tableRaw
	if err := tx.QueryRowContext(ctx, query,
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
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &tableRaw, nil
}

// patchTableImpl patches a table.
func (*Store) patchTableImpl(ctx context.Context, tx *sql.Tx, patch *api.TablePatch) (*tableRaw, error) {
	var tableRaw tableRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE tbl
		SET	type=$1, engine=$2, "collation"=$3, row_count=$4, data_size=$5, index_size=$6, data_free=$7, create_options=$8, comment=$9
		WHERE id = $10
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, type, engine, "collation", row_count, data_size, index_size, data_free, create_options, comment`,
		patch.Type,
		patch.Engine,
		patch.Collation,
		patch.RowCount,
		patch.DataSize,
		patch.IndexSize,
		patch.DataFree,
		patch.CreateOptions,
		patch.Comment,
		patch.ID,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("table ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &tableRaw, nil
}

func (*Store) findTableImpl(ctx context.Context, tx *sql.Tx, find *api.TableFind) ([]*tableRaw, error) {
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
func (*Store) deleteTableImpl(ctx context.Context, tx *sql.Tx, delete *api.TableDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM tbl WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
