package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

// dbExtensionRaw is the store model for an DBExtension.
// Fields have exactly the same meaning as DBExtension.
type dbExtensionRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	DatabaseID int

	// Domain specific fields
	Name        string
	Version     string
	Schema      string
	Description string
}

// toDBExtension creates an instance of DBExtension based on the dbExtensionRaw.
// This is intended to be called when we need to compose an DBExtension relationship.
func (raw *dbExtensionRaw) toDBExtension() *api.DBExtension {
	return &api.DBExtension{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:        raw.Name,
		Version:     raw.Version,
		Schema:      raw.Schema,
		Description: raw.Description,
	}
}

// FindDBExtension finds a list of dbExtension instances.
func (s *Store) FindDBExtension(ctx context.Context, find *api.DBExtensionFind) ([]*api.DBExtension, error) {
	dbExtensionRawList, err := s.findDBExtensionRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find dbExtension list with dbExtensionFind[%+v]", find)
	}
	var dbExtensionList []*api.DBExtension
	for _, raw := range dbExtensionRawList {
		dbExtension, err := s.composeDBExtension(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose dbExtension with dbExtensionRaw[%+v]", raw)
		}
		dbExtensionList = append(dbExtensionList, dbExtension)
	}
	return dbExtensionList, nil
}

type extensionKey struct {
	name   string
	schema string
}

// SetDBExtensionList sets the extensions for a database.
func (s *Store) SetDBExtensionList(ctx context.Context, schema *db.Schema, databaseID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	oldDBExtensionRawList, err := s.findDBExtensionImpl(ctx, tx, &api.DBExtensionFind{
		DatabaseID: &databaseID,
	})
	if err != nil {
		return FormatError(err)
	}

	deletes, creates := generateDBExtensionActions(oldDBExtensionRawList, schema.ExtensionList, databaseID)
	for _, d := range deletes {
		if err := s.deleteDBExtensionImpl(ctx, tx, d); err != nil {
			return err
		}
	}
	for _, c := range creates {
		if _, err := s.createDBExtensionImpl(ctx, tx, c); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// private functions.
func generateDBExtensionActions(oldDBExtensionRawList []*dbExtensionRaw, extensionList []db.Extension, databaseID int) ([]*api.DBExtensionDelete, []*api.DBExtensionCreate) {
	var newDBExtensionList []*api.DBExtensionCreate
	for _, dbExtension := range extensionList {
		newDBExtensionList = append(newDBExtensionList, &api.DBExtensionCreate{
			CreatorID:   api.SystemBotID,
			DatabaseID:  databaseID,
			Name:        dbExtension.Name,
			Version:     dbExtension.Version,
			Schema:      dbExtension.Schema,
			Description: dbExtension.Description,
		})
	}
	oldDBExtensionMap := make(map[extensionKey]*dbExtensionRaw)
	for _, e := range oldDBExtensionRawList {
		oldDBExtensionMap[extensionKey{name: e.Name, schema: e.Schema}] = e
	}
	newDBExtensionMap := make(map[extensionKey]*api.DBExtensionCreate)
	for _, e := range newDBExtensionList {
		newDBExtensionMap[extensionKey{name: e.Name, schema: e.Schema}] = e
	}

	var deletes []*api.DBExtensionDelete
	var creates []*api.DBExtensionCreate
	for _, oldValue := range oldDBExtensionRawList {
		k := extensionKey{name: oldValue.Name, schema: oldValue.Schema}
		newValue, ok := newDBExtensionMap[k]
		if !ok {
			deletes = append(deletes, &api.DBExtensionDelete{ID: oldValue.ID})
		} else if ok && (oldValue.Version != newValue.Version || oldValue.Description != newValue.Description) {
			deletes = append(deletes, &api.DBExtensionDelete{ID: oldValue.ID})
			creates = append(creates, newValue)
		}
	}
	for _, newValue := range newDBExtensionList {
		k := extensionKey{name: newValue.Name, schema: newValue.Schema}
		if _, ok := oldDBExtensionMap[k]; !ok {
			creates = append(creates, newValue)
		}
	}
	return deletes, creates
}

func (s *Store) composeDBExtension(ctx context.Context, raw *dbExtensionRaw) (*api.DBExtension, error) {
	dbExtension := raw.toDBExtension()

	creator, err := s.GetPrincipalByID(ctx, dbExtension.CreatorID)
	if err != nil {
		return nil, err
	}
	dbExtension.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, dbExtension.UpdaterID)
	if err != nil {
		return nil, err
	}
	dbExtension.Updater = updater

	database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: &dbExtension.DatabaseID})
	if err != nil {
		return nil, err
	}
	dbExtension.Database = database

	return dbExtension, nil
}

// findDBExtensionRaw retrieves a list of DBExtensions based on find.
func (s *Store) findDBExtensionRaw(ctx context.Context, find *api.DBExtensionFind) ([]*dbExtensionRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findDBExtensionImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// createDBExtensionImpl creates a new DBExtension.
func (*Store) createDBExtensionImpl(ctx context.Context, tx *Tx, create *api.DBExtensionCreate) (*dbExtensionRaw, error) {
	// Insert row into db_extension.
	query := `
		INSERT INTO db_extension (
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			database_id,
			name,
			version,
			schema,
			description
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, database_id, name, version, schema, description
	`
	var dbExtensionRaw dbExtensionRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatedTs,
		create.CreatorID,
		create.UpdatedTs,
		create.DatabaseID,
		create.Name,
		create.Version,
		create.Schema,
		create.Description,
	).Scan(
		&dbExtensionRaw.ID,
		&dbExtensionRaw.CreatorID,
		&dbExtensionRaw.CreatedTs,
		&dbExtensionRaw.UpdaterID,
		&dbExtensionRaw.UpdatedTs,
		&dbExtensionRaw.DatabaseID,
		&dbExtensionRaw.Name,
		&dbExtensionRaw.Version,
		&dbExtensionRaw.Schema,
		&dbExtensionRaw.Description,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &dbExtensionRaw, nil
}

func (*Store) findDBExtensionImpl(ctx context.Context, tx *Tx, find *api.DBExtensionFind) ([]*dbExtensionRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
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
			version,
			schema,
			description
		FROM db_extension
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY database_id, name ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into DBExtensionRawList.
	var dbExtensionRawList []*dbExtensionRaw
	for rows.Next() {
		var dbExtensionRaw dbExtensionRaw
		if err := rows.Scan(
			&dbExtensionRaw.ID,
			&dbExtensionRaw.CreatorID,
			&dbExtensionRaw.CreatedTs,
			&dbExtensionRaw.UpdaterID,
			&dbExtensionRaw.UpdatedTs,
			&dbExtensionRaw.DatabaseID,
			&dbExtensionRaw.Name,
			&dbExtensionRaw.Version,
			&dbExtensionRaw.Schema,
			&dbExtensionRaw.Description,
		); err != nil {
			return nil, FormatError(err)
		}

		dbExtensionRawList = append(dbExtensionRawList, &dbExtensionRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return dbExtensionRawList, nil
}

// deleteDBExtensionImpl permanently deletes DBExtensions from a database.
func (*Store) deleteDBExtensionImpl(ctx context.Context, tx *Tx, delete *api.DBExtensionDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM db_extension WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
