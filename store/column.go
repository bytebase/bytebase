package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

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

func generateColumnActions(oldColumnList []*api.Column, columnList []db.Column, databaseID, tableID int) ([]*api.ColumnDelete, []*api.ColumnCreate) {
	var columnCreateList []*api.ColumnCreate
	for _, column := range columnList {
		columnCreateList = append(columnCreateList, &api.ColumnCreate{
			CreatorID:    api.SystemBotID,
			DatabaseID:   databaseID,
			TableID:      tableID,
			Name:         column.Name,
			Position:     column.Position,
			Default:      column.Default,
			Nullable:     column.Nullable,
			Type:         column.Type,
			CharacterSet: column.CharacterSet,
			Collation:    column.Collation,
			Comment:      column.Comment,
		})
	}
	oldColumnMap := make(map[string]*api.Column)
	for _, c := range oldColumnList {
		oldColumnMap[c.Name] = c
	}
	newColumnMap := make(map[string]*api.ColumnCreate)
	for _, c := range columnCreateList {
		newColumnMap[c.Name] = c
	}

	var deletes []*api.ColumnDelete
	var creates []*api.ColumnCreate
	for _, oldValue := range oldColumnList {
		k := oldValue.Name
		newValue, ok := newColumnMap[k]
		if !ok {
			deletes = append(deletes, &api.ColumnDelete{ID: oldValue.ID})
		} else if ok && (oldValue.Position != newValue.Position || oldValue.Default != newValue.Default || oldValue.Nullable != newValue.Nullable || oldValue.Type != newValue.Type || oldValue.CharacterSet != newValue.CharacterSet || oldValue.Collation != newValue.Collation || oldValue.Comment != newValue.Comment) {
			deletes = append(deletes, &api.ColumnDelete{ID: oldValue.ID})
			creates = append(creates, newValue)
		}
	}
	for _, newValue := range columnCreateList {
		k := newValue.Name
		if _, ok := oldColumnMap[k]; !ok {
			creates = append(creates, newValue)
		}
	}
	return deletes, creates
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
	query := `
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
	`
	var column api.Column
	if err := tx.QueryRowContext(ctx, query,
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
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
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

	var column api.Column
	var defaultStr sql.NullString
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, `
		UPDATE col
		SET `+strings.Join(set, ", ")+`
		WHERE id = $2
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, name, position, "default", nullable, type, character_set, "collation", comment
	`,
		args...,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("column ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	if defaultStr.Valid {
		column.Default = &defaultStr.String
	}
	return &column, nil
}

// deleteColumnImpl deletes columns.
func deleteColumnImpl(ctx context.Context, tx *sql.Tx, delete *api.ColumnDelete) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM col WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
