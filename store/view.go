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
	_ api.ViewService = (*ViewService)(nil)
)

// ViewService represents a service for managing view.
type ViewService struct {
	l  *zap.Logger
	db *DB
}

// NewViewService returns a new instance of ViewService.
func NewViewService(logger *zap.Logger, db *DB) *ViewService {
	return &ViewService{l: logger, db: db}
}

// CreateView creates a new view.
func (s *ViewService) CreateView(ctx context.Context, create *api.ViewCreate) (*api.View, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	view, err := s.createView(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return view, nil
}

// FindViewList retrieves a list of views based on find.
func (s *ViewService) FindViewList(ctx context.Context, find *api.ViewFind) ([]*api.View, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findViewList(ctx, tx, find)
	if err != nil {
		return []*api.View{}, err
	}

	return list, nil
}

// FindView retrieves a single view based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *ViewService) FindView(ctx context.Context, find *api.ViewFind) (*api.View, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findViewList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d views with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// DeleteView deletes an existing view by ID.
func (s *ViewService) DeleteView(ctx context.Context, delete *api.ViewDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	err = deleteView(ctx, tx, delete)
	if err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// createView creates a new view.
func (s *ViewService) createView(ctx context.Context, tx *Tx, create *api.ViewCreate) (*api.View, error) {
	// Insert row into view.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO vw (
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
			name,
			definition,
			comment
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`+
		"RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, definition, comment"+`
	`,
		create.CreatorID,
		create.CreatedTs,
		create.CreatorID,
		create.UpdatedTs,
		create.DatabaseID,
		create.Name,
		create.Definition,
		create.Comment,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var view api.View
	if err := row.Scan(
		&view.ID,
		&view.CreatorID,
		&view.CreatedTs,
		&view.UpdaterID,
		&view.UpdatedTs,
		&view.DatabaseID,
		&view.Name,
		&view.Definition,
		&view.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	return &view, nil
}

func (s *ViewService) findViewList(ctx context.Context, tx *Tx, find *api.ViewFind) (_ []*api.View, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
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
			definition,
			comment
		FROM vw
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY database_id, name ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.View, 0)
	for rows.Next() {
		var view api.View
		if err := rows.Scan(
			&view.ID,
			&view.CreatorID,
			&view.CreatedTs,
			&view.UpdaterID,
			&view.UpdatedTs,
			&view.DatabaseID,
			&view.Name,
			&view.Definition,
			&view.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &view)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// deleteView permanently deletes views from a database.
func deleteView(ctx context.Context, tx *Tx, delete *api.ViewDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM vw WHERE database_id = ?`, delete.DatabaseID); err != nil {
		return FormatError(err)
	}
	return nil
}
