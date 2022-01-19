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
	_ api.SavedQueryService = (*SavedQueryService)(nil)
)

// SavedQueryService represents a service for managing saved_query.
type SavedQueryService struct {
	l  *zap.Logger
	db *DB
}

// NewSavedQueryService returns a new saved_query of SavedQueryService.
func NewSavedQueryService(logger *zap.Logger, db *DB) *SavedQueryService {
	return &SavedQueryService{l: logger, db: db}
}

// CreateSavedQuery creates a new saved_query.
func (s *SavedQueryService) CreateSavedQuery(ctx context.Context, create *api.SavedQueryCreate) (*api.SavedQuery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	savedQuery, err := createSavedQuery(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return savedQuery, nil
}

// PatchSavedQuery updates an existing saved_query by ID.
func (s *SavedQueryService) PatchSavedQuery(ctx context.Context, patch *api.SavedQueryPatch) (*api.SavedQuery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	savedQuery, err := patchSavedQuery(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return savedQuery, nil
}

// FindSavedQueryList retrieves a list of saved_queries based on find.
func (s *SavedQueryService) FindSavedQueryList(ctx context.Context, find *api.SavedQueryFind) ([]*api.SavedQuery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findSavedQueryList(ctx, tx, find)
	if err != nil {
		return []*api.SavedQuery{}, err
	}

	return list, nil
}

// FindSavedQuery retrieves a single saved_query based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *SavedQueryService) FindSavedQuery(ctx context.Context, find *api.SavedQueryFind) (*api.SavedQuery, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := findSavedQueryList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d saved_queries with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// DeleteSavedQuery deletes an existing saved_query by ID.
// Returns ENOTFOUND if saved_query does not exist.
func (s *SavedQueryService) DeleteSavedQuery(ctx context.Context, delete *api.SavedQueryDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteSavedQuery(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createSavedQuery creates a new saved_query.
func createSavedQuery(ctx context.Context, tx *Tx, create *api.SavedQueryCreate) (*api.SavedQuery, error) {
	row, err := tx.QueryContext(ctx, `
		INSERT INTO saved_query (
			creator_id,
			updater_id,
			name,
			statement
		)
		VALUES (?, ?, ?, ?)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, statement
	`,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Statement,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var savedQuery api.SavedQuery
	if err := row.Scan(
		&savedQuery.ID,
		&savedQuery.CreatorID,
		&savedQuery.CreatedTs,
		&savedQuery.UpdaterID,
		&savedQuery.UpdatedTs,
		&savedQuery.Name,
		&savedQuery.Statement,
	); err != nil {
		return nil, FormatError(err)
	}

	return &savedQuery, nil
}

// patchSavedQuery creates a new saved_query.
func patchSavedQuery(ctx context.Context, tx *Tx, patch *api.SavedQueryPatch) (*api.SavedQuery, error) {
	set, args := []string{"updater_id = ?"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, api.RowStatus(*v))
	}
	if v := patch.Statement; v != nil {
		set, args = append(set, "statement = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	row, err := tx.QueryContext(ctx, `
		UPDATE saved_query
		SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, name, statement
	`,
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var savedQuery api.SavedQuery
		if err := row.Scan(
			&savedQuery.ID,
			&savedQuery.CreatorID,
			&savedQuery.CreatedTs,
			&savedQuery.UpdaterID,
			&savedQuery.UpdatedTs,
			&savedQuery.Name,
			&savedQuery.Statement,
		); err != nil {
			return nil, FormatError(err)
		}

		return &savedQuery, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("saved query ID not found: %d", patch.ID)}
}

func findSavedQueryList(ctx context.Context, tx *Tx, find *api.SavedQueryFind) (_ []*api.SavedQuery, err error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, "creator_id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
		    id,
		    creator_id,
		    created_ts,
		    updater_id,
		    updated_ts,
		    name,
		    statement
		FROM saved_query
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	list := make([]*api.SavedQuery, 0)
	for rows.Next() {
		var savedQuery api.SavedQuery
		if err := rows.Scan(
			&savedQuery.ID,
			&savedQuery.CreatorID,
			&savedQuery.CreatedTs,
			&savedQuery.UpdaterID,
			&savedQuery.UpdatedTs,
			&savedQuery.Name,
			&savedQuery.Statement,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &savedQuery)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// deleteSavedQuery permanently deletes a saved_query by ID.
func deleteSavedQuery(ctx context.Context, tx *Tx, delete *api.SavedQueryDelete) error {
	result, err := tx.ExecContext(ctx, `DELETE FROM saved_query WHERE id = ?`, delete.ID)
	if err != nil {
		return FormatError(err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("saved query ID not found: %d", delete.ID)}
	}

	return nil
}
