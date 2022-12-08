package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// dbSchemaRaw is the store model for a database schema.
// Fields have exactly the same meaning as DBSchema.
type dbSchemaRaw struct {
	ID int

	// Standard fields

	// Related fields
	DatabaseID int

	// Domain specific fields
	Metadata string
	RawDump  string
}

func (raw *dbSchemaRaw) toDBSchema() *api.DBSchema {
	return &api.DBSchema{
		ID: raw.ID,

		// Standard fields

		// Related fields
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Metadata: raw.Metadata,
		RawDump:  raw.RawDump,
	}
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, databaseID int) (*api.DBSchema, error) {
	cachedDBSchema := &api.DBSchema{}
	ok, err := s.cache.FindCache(principalCacheNamespace, databaseID, cachedDBSchema)
	if err != nil {
		return nil, err
	}
	if ok {
		return cachedDBSchema, nil
	}

	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, databaseID)

	rows, err := s.db.db.QueryContext(ctx, `
		SELECT
			id,
			database_id,
			metadata,
			raw_dump
		FROM db_schema
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var list []*dbSchemaRaw
	for rows.Next() {
		var raw dbSchemaRaw
		if err := rows.Scan(
			&raw.ID,
			&raw.DatabaseID,
			&raw.Metadata,
			&raw.RawDump,
		); err != nil {
			return nil, FormatError(err)
		}
		list = append(list, &raw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d database schema for database %d, expect 1", len(list), databaseID)}
	}

	dbSchema := list[0].toDBSchema()
	if err := s.cache.UpsertCache(schemaCacheNamespace, databaseID, dbSchema); err != nil {
		return nil, err
	}
	return dbSchema, nil
}

// UpsertDBSchema upserts a database schema.
func (s *Store) UpsertDBSchema(ctx context.Context, upsert api.DBSchemaUpsert) (*api.DBSchema, error) {
	query := `
		INSERT INTO db_schema (
			creator_id,
			updater_id,
			database_id,
			metadata,
			raw_dump
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(database_id) DO UPDATE SET
			metadata = EXCLUDED.metadata,
			raw_dump = EXCLUDED.raw_dump
		RETURNING id, database_id, metadata, raw_dump
	`
	var raw dbSchemaRaw
	if err := s.db.db.QueryRowContext(ctx, query,
		upsert.UpdatorID,
		upsert.UpdatorID,
		upsert.DatabaseID,
		upsert.Metadata,
		upsert.RawDump,
	).Scan(
		&raw.ID,
		&raw.DatabaseID,
		&raw.Metadata,
		&raw.RawDump,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	dbSchema := raw.toDBSchema()
	if err := s.cache.UpsertCache(schemaCacheNamespace, upsert.DatabaseID, dbSchema); err != nil {
		return nil, err
	}
	return dbSchema, nil
}

// DeleteDBSchema deletes the schema for a database.
func (s *Store) DeleteDBSchema(ctx context.Context, databaseID int) (*api.DBSchema, error) {
	if _, err := s.db.db.ExecContext(ctx, "DELETE FROM db_schema WHERE database_id = $1", databaseID); err != nil {
		return nil, FormatError(err)
	}
	// Invalidate cache.
	s.cache.DeleteCache(schemaCacheNamespace, databaseID)
	return nil, nil
}
