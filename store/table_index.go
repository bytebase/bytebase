package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	_ api.IndexService = (*IndexService)(nil)
)

// IndexService represents a service for managing index.
type IndexService struct {
	l  *zap.Logger
	db *DB
}

// NewIndexService returns a new instance of IndexService.
func NewIndexService(logger *zap.Logger, db *DB) *IndexService {
	return &IndexService{l: logger, db: db}
}

// CreateIndex creates a new index.
func (s *IndexService) CreateIndex(ctx context.Context, create *api.IndexCreate) (*api.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	index, err := s.createIndex(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return index, nil
}

// FindIndexList retrieves a list of indexs based on find.
func (s *IndexService) FindIndexList(ctx context.Context, find *api.IndexFind) ([]*api.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findIndexList(ctx, tx, find)
	if err != nil {
		return []*api.Index{}, err
	}

	return list, nil
}

// FindIndex retrieves a single index based on find.
// Returns ENOTFOUND if no matching record.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *IndexService) FindIndex(ctx context.Context, find *api.IndexFind) (*api.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findIndexList(ctx, tx, find)
	if err != nil {
		return nil, err
	} else if len(list) == 0 {
		return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("index not found: %+v", find)}
	} else if len(list) > 1 {
		return nil, &bytebase.Error{Code: bytebase.ECONFLICT, Message: fmt.Sprintf("found %d indexs with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// PatchIndex updates an existing index by ID.
// Returns ENOTFOUND if index does not exist.
func (s *IndexService) PatchIndex(ctx context.Context, patch *api.IndexPatch) (*api.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	index, err := s.patchIndex(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return index, nil
}

// createIndex creates a new index.
func (s *IndexService) createIndex(ctx context.Context, tx *Tx, create *api.IndexCreate) (*api.Index, error) {
	// Insert row into index.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO idx (
			creator_id,
			updater_id,
			database_id,
			table_id,
			sync_status,
			last_successful_sync_ts,
			name,
			expression,
			position,
			`+"`type`,"+`
			`+"`unique`,"+`
			visible,
			comment
		)
		VALUES (?, ?, ?, ?, 'OK', (strftime('%s', 'now')), ?, ?, ?, ?, ?, ?, ?)`+
		"RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, sync_status, last_successful_sync_ts, name, expression, position, `type`, `unique`, visible, comment"+`
	`,
		create.CreatorId,
		create.CreatorId,
		create.DatabaseId,
		create.TableId,
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
		&index.CreatorId,
		&index.CreatedTs,
		&index.UpdaterId,
		&index.UpdatedTs,
		&index.DatabaseId,
		&index.TableId,
		&index.SyncStatus,
		&index.LastSuccessfulSyncTs,
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

func (s *IndexService) findIndexList(ctx context.Context, tx *Tx, find *api.IndexFind) (_ []*api.Index, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.DatabaseId; v != nil {
		where, args = append(where, "database_id = ?"), append(args, *v)
	}
	if v := find.TableId; v != nil {
		where, args = append(where, "table_id = ?"), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
	}
	if v := find.Expression; v != nil {
		where, args = append(where, "expression = ?"), append(args, *v)
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
			sync_status,
			last_successful_sync_ts,
			name,
			expression,
			position,
			`+"`type`,"+`
			`+"`unique`,"+`
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

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Index, 0)
	for rows.Next() {
		var index api.Index
		if err := rows.Scan(
			&index.ID,
			&index.CreatorId,
			&index.CreatedTs,
			&index.UpdaterId,
			&index.UpdatedTs,
			&index.DatabaseId,
			&index.TableId,
			&index.SyncStatus,
			&index.LastSuccessfulSyncTs,
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

		list = append(list, &index)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// patchIndex updates a index by ID. Returns the new state of the index after update.
func (s *IndexService) patchIndex(ctx context.Context, tx *Tx, patch *api.IndexPatch) (*api.Index, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterId}
	if v := patch.SyncStatus; v != nil {
		set, args = append(set, "sync_status = ?"), append(args, api.SyncStatus(*v))
	}
	if v := patch.LastSuccessfulSyncTs; v != nil {
		set, args = append(set, "last_successful_sync_ts = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, `
		UPDATE idx
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?`+
		"RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, table_id, sync_status, last_successful_sync_ts, name, expression, position, `type`, `unique`, visible, comment"+`
	`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var index api.Index
		if err := row.Scan(
			&index.ID,
			&index.CreatorId,
			&index.CreatedTs,
			&index.UpdaterId,
			&index.UpdatedTs,
			&index.DatabaseId,
			&index.TableId,
			&index.SyncStatus,
			&index.LastSuccessfulSyncTs,
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

	return nil, &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("index ID not found: %d", patch.ID)}
}
