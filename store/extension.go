package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// extensionRaw is the store model for an Extension.
// Fields have exactly the same meanings as Extension.
type extensionRaw struct {
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

// toExtension creates an instance of Extension based on the extensionRaw.
// This is intended to be called when we need to compose an Extension relationship.
func (raw *extensionRaw) toExtension() *api.Extension {
	return &api.Extension{
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

// CreateExtension creates an instance of Extension
func (s *Store) CreateExtension(ctx context.Context, create *api.ExtensionCreate) (*api.Extension, error) {
	if s.db.mode == common.ReleaseModeProd {
		return nil, nil
	}
	extensionRaw, err := s.createExtensionRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Extension with ExtensionCreate[%+v], error[%w]", create, err)
	}
	extension, err := s.composeExtension(ctx, extensionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Extension with extensionRaw[%+v], error[%w]", extensionRaw, err)
	}
	return extension, nil
}

// FindExtension finds a list of Extension instances
func (s *Store) FindExtension(ctx context.Context, find *api.ExtensionFind) ([]*api.Extension, error) {
	if s.db.mode == common.ReleaseModeProd {
		return nil, nil
	}
	extensionRawList, err := s.findExtensionRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Extension list with ExtensionFind[%+v], error[%w]", find, err)
	}
	var extensionList []*api.Extension
	for _, raw := range extensionRawList {
		extension, err := s.composeExtension(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Extension with extensionRaw[%+v], error[%w]", raw, err)
		}
		extensionList = append(extensionList, extension)
	}
	return extensionList, nil
}

// DeleteExtension deletes an existing extension by ID.
func (s *Store) DeleteExtension(ctx context.Context, delete *api.ExtensionDelete) error {
	if s.db.mode == common.ReleaseModeProd {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteExtensionImpl(ctx, tx.PTx, delete); err != nil {
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

func (s *Store) composeExtension(ctx context.Context, raw *extensionRaw) (*api.Extension, error) {
	extension := raw.toExtension()

	creator, err := s.GetPrincipalByID(ctx, extension.CreatorID)
	if err != nil {
		return nil, err
	}
	extension.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, extension.UpdaterID)
	if err != nil {
		return nil, err
	}
	extension.Updater = updater

	database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: &extension.DatabaseID})
	if err != nil {
		return nil, err
	}
	extension.Database = database

	return extension, nil
}

// createExtensionRaw creates a new extension.
func (s *Store) createExtensionRaw(ctx context.Context, create *api.ExtensionCreate) (*extensionRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	extension, err := s.createExtensionImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return extension, nil
}

// findExtensionRaw retrieves a list of extensions based on find.
func (s *Store) findExtensionRaw(ctx context.Context, find *api.ExtensionFind) ([]*extensionRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findExtensionImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// createExtensionImpl creates a new extension.
func (s *Store) createExtensionImpl(ctx context.Context, tx *sql.Tx, create *api.ExtensionCreate) (*extensionRaw, error) {
	// Insert row into extension.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO extension (
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
	var extensionRaw extensionRaw
	if err := row.Scan(
		&extensionRaw.ID,
		&extensionRaw.CreatorID,
		&extensionRaw.CreatedTs,
		&extensionRaw.UpdaterID,
		&extensionRaw.UpdatedTs,
		&extensionRaw.DatabaseID,
		&extensionRaw.Name,
		&extensionRaw.Version,
		&extensionRaw.Schema,
		&extensionRaw.Description,
	); err != nil {
		return nil, FormatError(err)
	}

	return &extensionRaw, nil
}

func (s *Store) findExtensionImpl(ctx context.Context, tx *sql.Tx, find *api.ExtensionFind) ([]*extensionRaw, error) {
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
		FROM extension
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY database_id, name ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into extensionRawList.
	var extensionRawList []*extensionRaw
	for rows.Next() {
		var extensionRaw extensionRaw
		if err := rows.Scan(
			&extensionRaw.ID,
			&extensionRaw.CreatorID,
			&extensionRaw.CreatedTs,
			&extensionRaw.UpdaterID,
			&extensionRaw.UpdatedTs,
			&extensionRaw.DatabaseID,
			&extensionRaw.Name,
			&extensionRaw.Version,
			&extensionRaw.Schema,
			&extensionRaw.Description,
		); err != nil {
			return nil, FormatError(err)
		}

		extensionRawList = append(extensionRawList, &extensionRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return extensionRawList, nil
}

// deleteExtensionImpl permanently deletes extensions from a database.
func deleteExtensionImpl(ctx context.Context, tx *sql.Tx, delete *api.ExtensionDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM extension WHERE database_id = $1`, delete.DatabaseID); err != nil {
		return FormatError(err)
	}
	return nil
}
