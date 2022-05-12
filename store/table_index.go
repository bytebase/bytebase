package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// CreateIndex creates a new index.
func (s *Store) CreateIndex(ctx context.Context, create *api.IndexCreate) (*api.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	index, err := s.createIndexImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return index, nil
}

// FindIndex retrieves a list of indexes based on find.
func (s *Store) FindIndex(ctx context.Context, find *api.IndexFind) ([]*api.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findIndexImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// GetIndex retrieves a single index based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) GetIndex(ctx context.Context, find *api.IndexFind) (*api.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findIndexImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d indexes with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// createIndexImpl creates a new index.
func (s *Store) createIndexImpl(ctx context.Context, tx *sql.Tx, create *api.IndexCreate) (*api.Index, error) {
	// Insert row into index.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO idx (
			creator_id,
			updater_id,
			database_id,
			table_id,
			name,
			expression,
			position,
			type,
			"unique",
			visible,
			comment
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, name, expression, position, type, "unique", visible, comment
	`,
		create.CreatorID,
		create.CreatorID,
		create.DatabaseID,
		create.TableID,
		create.Name,
		create.Expression,
		create.Position,
		create.Type,
		create.Unique,
		create.Visible,
		create.Comment,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var index api.Index
	if err := row.Scan(
		&index.ID,
		&index.CreatorID,
		&index.CreatedTs,
		&index.UpdaterID,
		&index.UpdatedTs,
		&index.DatabaseID,
		&index.TableID,
		&index.Name,
		&index.Expression,
		&index.Position,
		&index.Type,
		&index.Unique,
		&index.Visible,
		&index.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	return &index, nil
}

func (s *Store) findIndexImpl(ctx context.Context, tx *sql.Tx, find *api.IndexFind) ([]*api.Index, error) {
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
	if v := find.Expression; v != nil {
		where, args = append(where, fmt.Sprintf("expression = $%d", len(args)+1)), append(args, *v)
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
			expression,
			position,
			type,
			"unique",
			visible,
			comment
		FROM idx
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY database_id, table_id, CASE name WHEN 'PRIMARY' THEN 1 ELSE 2 END, name ASC, position ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into indexList.
	var indexList []*api.Index
	for rows.Next() {
		var index api.Index
		if err := rows.Scan(
			&index.ID,
			&index.CreatorID,
			&index.CreatedTs,
			&index.UpdaterID,
			&index.UpdatedTs,
			&index.DatabaseID,
			&index.TableID,
			&index.Name,
			&index.Expression,
			&index.Position,
			&index.Type,
			&index.Unique,
			&index.Visible,
			&index.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		indexList = append(indexList, &index)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return indexList, nil
}
