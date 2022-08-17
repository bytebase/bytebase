package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pkg/errors"
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

// FindView finds a list of View instances.
func (s *Store) FindView(ctx context.Context, find *api.ViewFind) ([]*api.View, error) {
	viewRawList, err := s.findViewRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find View list with ViewFind[%+v]", find)
	}
	var viewList []*api.View
	for _, raw := range viewRawList {
		view, err := s.composeView(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose View with viewRaw[%+v]", raw)
		}
		viewList = append(viewList, view)
	}
	return viewList, nil
}

// SetViewList sets the views for a database.
func (s *Store) SetViewList(ctx context.Context, schema *db.Schema, databaseID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	oldViewRawList, err := s.findViewImpl(ctx, tx.PTx, &api.ViewFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return FormatError(err)
	}

	deletes, creates := generateViewActions(oldViewRawList, schema.ViewList, databaseID)
	for _, d := range deletes {
		if err := s.deleteViewImpl(ctx, tx.PTx, d); err != nil {
			return err
		}
	}
	for _, c := range creates {
		if _, err := s.createViewImpl(ctx, tx.PTx, c); err != nil {
			return err
		}
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

//
// private functions.
//
func generateViewActions(oldViewRawList []*viewRaw, viewList []db.View, databaseID int) ([]*api.ViewDelete, []*api.ViewCreate) {
	var viewCreateList []*api.ViewCreate
	for _, view := range viewList {
		viewCreateList = append(viewCreateList, &api.ViewCreate{
			CreatorID:  api.SystemBotID,
			CreatedTs:  view.CreatedTs,
			UpdatedTs:  view.UpdatedTs,
			DatabaseID: databaseID,
			Name:       view.Name,
			Definition: view.Definition,
			Comment:    view.Comment,
		})
	}
	oldViewMap := make(map[string]*viewRaw)
	for _, v := range oldViewRawList {
		oldViewMap[v.Name] = v
	}
	newViewMap := make(map[string]*api.ViewCreate)
	for _, v := range viewCreateList {
		newViewMap[v.Name] = v
	}

	var deletes []*api.ViewDelete
	var creates []*api.ViewCreate
	for _, oldValue := range oldViewRawList {
		k := oldValue.Name
		newValue, ok := newViewMap[k]
		if !ok {
			deletes = append(deletes, &api.ViewDelete{ID: oldValue.ID})
		} else if ok && (oldValue.Definition != newValue.Definition || oldValue.Comment != newValue.Comment) {
			deletes = append(deletes, &api.ViewDelete{ID: oldValue.ID})
			creates = append(creates, newValue)
		}
	}
	for _, newValue := range viewCreateList {
		k := newValue.Name
		if _, ok := oldViewMap[k]; !ok {
			creates = append(creates, newValue)
		}
	}
	return deletes, creates
}

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
func (*Store) createViewImpl(ctx context.Context, tx *sql.Tx, create *api.ViewCreate) (*viewRaw, error) {
	// Insert row into view.
	query := `
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)` +
		"RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, definition, comment" + `
	`
	var viewRaw viewRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatedTs,
		create.CreatorID,
		create.UpdatedTs,
		create.DatabaseID,
		create.Name,
		create.Definition,
		create.Comment,
	).Scan(
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
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &viewRaw, nil
}

func (*Store) findViewImpl(ctx context.Context, tx *sql.Tx, find *api.ViewFind) ([]*viewRaw, error) {
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
func (*Store) deleteViewImpl(ctx context.Context, tx *sql.Tx, delete *api.ViewDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM vw WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
