package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
)

// viewRaw is the store model for an View.
// Fields have exactly the same meanings as View.
type viewRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name       string
	Definition string
	Comment    string
}

// toView creates an instance of View based on the viewRaw.
// This is intended to be called when we need to compose an View relationship.
func (raw *viewRaw) toView() *api.View {
	return &api.View{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:       raw.Name,
		Definition: raw.Definition,
		Comment:    raw.Comment,
	}
}

// CreateView creates an instance of View
func (s *Store) CreateView(ctx context.Context, create *api.ViewCreate) (*api.View, error) {
	viewRaw, err := s.createViewRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create View with ViewCreate[%+v], error[%w]", create, err)
	}
	view, err := s.composeView(ctx, viewRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose View with viewRaw[%+v], error[%w]", viewRaw, err)
	}
	return view, nil
}

// FindView finds a list of View instances
func (s *Store) FindView(ctx context.Context, find *api.ViewFind) ([]*api.View, error) {
	viewRawList, err := s.findViewRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find View list with ViewFind[%+v], error[%w]", find, err)
	}
	var viewList []*api.View
	for _, raw := range viewRawList {
		view, err := s.composeView(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose View with viewRaw[%+v], error[%w]", raw, err)
		}
		viewList = append(viewList, view)
	}
	return viewList, nil
}

// DeleteView deletes an existing view by ID.
func (s *Store) DeleteView(ctx context.Context, delete *api.ViewDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteViewImpl(ctx, tx.PTx, delete); err != nil {
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

func (s *Store) composeView(ctx context.Context, raw *viewRaw) (*api.View, error) {
	view := raw.toView()

	creator, err := s.GetPrincipalByID(ctx, view.CreatorID)
	if err != nil {
		return nil, err
	}
	view.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, view.UpdaterID)
	if err != nil {
		return nil, err
	}
	view.Updater = updater

	database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: &view.DatabaseID})
	if err != nil {
		return nil, err
	}
	view.Database = database

	return view, nil
}

// createViewRaw creates a new view.
func (s *Store) createViewRaw(ctx context.Context, create *api.ViewCreate) (*viewRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	view, err := s.createViewImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return view, nil
}

// findViewRaw retrieves a list of views based on find.
func (s *Store) findViewRaw(ctx context.Context, find *api.ViewFind) ([]*viewRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findViewImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// createViewImpl creates a new view.
func (s *Store) createViewImpl(ctx context.Context, tx *sql.Tx, create *api.ViewCreate) (*viewRaw, error) {
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`+
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
	var viewRaw viewRaw
	if err := row.Scan(
		&viewRaw.ID,
		&viewRaw.CreatorID,
		&viewRaw.CreatedTs,
		&viewRaw.UpdaterID,
		&viewRaw.UpdatedTs,
		&viewRaw.DatabaseID,
		&viewRaw.Name,
		&viewRaw.Definition,
		&viewRaw.Comment,
	); err != nil {
		return nil, FormatError(err)
	}

	return &viewRaw, nil
}

func (s *Store) findViewImpl(ctx context.Context, tx *sql.Tx, find *api.ViewFind) ([]*viewRaw, error) {
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

	// Iterate over result set and deserialize rows into viewRawList.
	var viewRawList []*viewRaw
	for rows.Next() {
		var viewRaw viewRaw
		if err := rows.Scan(
			&viewRaw.ID,
			&viewRaw.CreatorID,
			&viewRaw.CreatedTs,
			&viewRaw.UpdaterID,
			&viewRaw.UpdatedTs,
			&viewRaw.DatabaseID,
			&viewRaw.Name,
			&viewRaw.Definition,
			&viewRaw.Comment,
		); err != nil {
			return nil, FormatError(err)
		}

		viewRawList = append(viewRawList, &viewRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return viewRawList, nil
}

// deleteViewImpl permanently deletes views from a database.
func deleteViewImpl(ctx context.Context, tx *sql.Tx, delete *api.ViewDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM vw WHERE database_id = $1`, delete.DatabaseID); err != nil {
		return FormatError(err)
	}
	return nil
}
