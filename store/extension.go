package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
)

// dbExtensionRaw is the store model for an DBExtension.
// Fields have exactly the same meanings as DBExtension.
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

// CreateDBExtension creates an instance of DBExtension
func (s *Store) CreateDBExtension(ctx context.Context, create *api.DBExtensionCreate) (*api.DBExtension, error) {
	dbExtensionRaw, err := s.createDBExtensionRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create dbExtension with dbExtensionCreate[%+v], error[%w]", create, err)
	}
	dbExtension, err := s.composeDBExtension(ctx, dbExtensionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose dbExtension with dbExtensionRaw[%+v], error[%w]", dbExtensionRaw, err)
	}
	return dbExtension, nil
}

// FindDBExtension finds a list of dbExtension instances
func (s *Store) FindDBExtension(ctx context.Context, find *api.DBExtensionFind) ([]*api.DBExtension, error) {
	dbExtensionRawList, err := s.findDBExtensionRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find dbExtension list with dbExtensionFind[%+v], error[%w]", find, err)
	}
	var dbExtensionList []*api.DBExtension
	for _, raw := range dbExtensionRawList {
		dbExtension, err := s.composeDBExtension(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose dbExtension with dbExtensionRaw[%+v], error[%w]", raw, err)
		}
		dbExtensionList = append(dbExtensionList, dbExtension)
	}
	return dbExtensionList, nil
}

// DeleteDBExtension deletes an existing dbExtension by ID.
func (s *Store) DeleteDBExtension(ctx context.Context, delete *api.DBExtensionDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteDBExtensionImpl(ctx, tx.PTx, delete); err != nil {
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

// createDBExtensionRaw creates a new DBExtension.
func (s *Store) createDBExtensionRaw(ctx context.Context, create *api.DBExtensionCreate) (*dbExtensionRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	dbExtension, err := s.createDBExtensionImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return dbExtension, nil
}

// findDBExtensionRaw retrieves a list of DBExtensions based on find.
func (s *Store) findDBExtensionRaw(ctx context.Context, find *api.DBExtensionFind) ([]*dbExtensionRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findDBExtensionImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// createDBExtensionImpl creates a new DBExtension.
func (s *Store) createDBExtensionImpl(ctx context.Context, tx *sql.Tx, create *api.DBExtensionCreate) (*dbExtensionRaw, error) {
	// Insert row into db_extension.
	row, err := tx.QueryContext(ctx, `
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
	`,
		create.CreatorID,
		create.CreatedTs,
		create.CreatorID,
		create.UpdatedTs,
		create.DatabaseID,
		create.Name,
		create.Version,
		create.Schema,
		create.Description,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var dbExtensionRaw dbExtensionRaw
	if err := row.Scan(
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

	return &dbExtensionRaw, nil
}

func (s *Store) findDBExtensionImpl(ctx context.Context, tx *sql.Tx, find *api.DBExtensionFind) ([]*dbExtensionRaw, error) {
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
func deleteDBExtensionImpl(ctx context.Context, tx *sql.Tx, delete *api.DBExtensionDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM db_extension WHERE database_id = $1`, delete.DatabaseID); err != nil {
		return FormatError(err)
	}
	return nil
}
