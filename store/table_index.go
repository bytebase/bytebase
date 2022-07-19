package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

// FindIndex retrieves a list of indices based on find.
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

type indexKey struct {
	name     string
	position int
}

func generateIndexActions(oldIndexList []*api.Index, indexList []db.Index, databaseID, tableID int) ([]*api.IndexDelete, []*api.IndexCreate) {
	var indexCreateList []*api.IndexCreate
	for _, index := range indexList {
		indexCreateList = append(indexCreateList, &api.IndexCreate{
			CreatorID:  api.SystemBotID,
			DatabaseID: databaseID,
			TableID:    tableID,
			Name:       index.Name,
			Expression: index.Expression,
			Position:   index.Position,
			Type:       index.Type,
			Unique:     index.Unique,
			Primary:    index.Primary,
			Visible:    index.Visible,
			Comment:    index.Comment,
		})
	}
	oldIndexMap := make(map[indexKey]*api.Index)
	for _, index := range oldIndexList {
		oldIndexMap[indexKey{index.Name, index.Position}] = index
	}
	newIndexMap := make(map[indexKey]*api.IndexCreate)
	for _, index := range indexCreateList {
		newIndexMap[indexKey{index.Name, index.Position}] = index
	}

	var deletes []*api.IndexDelete
	var creates []*api.IndexCreate
	for _, oldValue := range oldIndexList {
		k := indexKey{oldValue.Name, oldValue.Position}
		newValue, ok := newIndexMap[k]
		if !ok {
			deletes = append(deletes, &api.IndexDelete{ID: oldValue.ID})
		} else if ok && (oldValue.Expression != newValue.Expression || oldValue.Position != newValue.Position || oldValue.Type != newValue.Type || oldValue.Unique != newValue.Unique || oldValue.Primary != newValue.Primary || oldValue.Visible != newValue.Visible || oldValue.Comment != newValue.Comment) {
			deletes = append(deletes, &api.IndexDelete{ID: oldValue.ID})
			creates = append(creates, newValue)
		}
	}
	for _, newValue := range indexCreateList {
		k := indexKey{newValue.Name, newValue.Position}
		if _, ok := oldIndexMap[k]; !ok {
			creates = append(creates, newValue)
		}
	}
	// The ordering of creates and deletes are not consistently produced because of maps. We need to produce a consistent output
	// for callers such as testing.
	sort.Slice(deletes, func(i, j int) bool {
		return deletes[i].ID < deletes[j].ID
	})
	sort.Slice(creates, func(i, j int) bool {
		return creates[i].Name < creates[j].Name || (creates[i].Name == creates[j].Name && creates[i].Position < creates[j].Position)
	})

	return deletes, creates
}

// createIndexImpl creates a new index.
func (s *Store) createIndexImpl(ctx context.Context, tx *sql.Tx, create *api.IndexCreate) (*api.Index, error) {
	if s.db.mode == common.ReleaseModeDev {
		// Insert row into index.
		query := `
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
			"primary",
			visible,
			comment
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, name, expression, position, type, "unique", "primary", visible, comment
	`
		var index api.Index
		if err := tx.QueryRowContext(ctx, query,
			create.CreatorID,
			create.CreatorID,
			create.DatabaseID,
			create.TableID,
			create.Name,
			create.Expression,
			create.Position,
			create.Type,
			create.Unique,
			create.Primary,
			create.Visible,
			create.Comment,
		).Scan(
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
			&index.Primary,
			&index.Visible,
			&index.Comment,
		); err != nil {
			if err == sql.ErrNoRows {
				return nil, common.FormatDBErrorEmptyRowWithQuery(query)
			}
			return nil, FormatError(err)
		}

		return &index, nil
	}
	// Insert row into index.
	query := `
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
	`
	var index api.Index
	if err := tx.QueryRowContext(ctx, query,
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
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
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

	if s.db.mode == common.ReleaseModeDev {
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
				"primary",
				visible,
				comment
			FROM idx
			WHERE `+strings.Join(where, " AND ")+`
			ORDER BY database_id, table_id, CASE WHEN "primary" THEN 1 ELSE 2 END, name ASC, position ASC`,
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
				&index.Primary,
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

// deleteIndexImpl deletes an index.
func deleteIndexImpl(ctx context.Context, tx *sql.Tx, delete *api.IndexDelete) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM idx WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
