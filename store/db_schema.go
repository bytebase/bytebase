package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DBSchema is the API message for database schema.
type DBSchema struct {
	Metadata *storepb.DatabaseMetadata
	RawDump  string
}

// DBSchemaUpsert is the API message for creating a database schema.
type DBSchemaUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Related fields
	DatabaseID int

	// Domain specific fields
	Metadata *storepb.DatabaseMetadata
	RawDump  string
}

type dbSchemaRaw struct {
	Metadata string
	RawDump  string
}

func (raw *dbSchemaRaw) toDBSchema() (*DBSchema, error) {
	var databaseSchema storepb.DatabaseMetadata
	decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := decoder.Unmarshal([]byte(raw.Metadata), &databaseSchema); err != nil {
		return nil, err
	}
	return &DBSchema{
		Metadata: &databaseSchema,
		RawDump:  raw.RawDump,
	}, nil
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, databaseID int) (*DBSchema, error) {
	cachedDBSchema := &DBSchema{}
	ok, err := s.cache.FindCache(schemaCacheNamespace, databaseID, cachedDBSchema)
	if err != nil {
		return nil, err
	}
	if ok {
		return cachedDBSchema, nil
	}

	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, databaseID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var raw dbSchemaRaw
	if err := tx.QueryRowContext(ctx, `
		SELECT
			metadata,
			raw_dump
		FROM db_schema
		WHERE `+strings.Join(where, " AND "),
		args...,
	).Scan(
		&raw.Metadata,
		&raw.RawDump,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	dbSchema, err := raw.toDBSchema()
	if err != nil {
		return nil, err
	}
	if err := s.cache.UpsertCache(schemaCacheNamespace, databaseID, dbSchema); err != nil {
		return nil, err
	}
	return dbSchema, nil
}

// UpsertDBSchema upserts a database schema.
func (s *Store) UpsertDBSchema(ctx context.Context, upsert DBSchemaUpsert) error {
	metadataBytes, err := protojson.Marshal(upsert.Metadata)
	if err != nil {
		return err
	}

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
		RETURNING metadata, raw_dump
	`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	var raw dbSchemaRaw
	if err := tx.QueryRowContext(ctx, query,
		upsert.UpdaterID,
		upsert.UpdaterID,
		upsert.DatabaseID,
		metadataBytes,
		upsert.RawDump,
	).Scan(
		&raw.Metadata,
		&raw.RawDump,
	); err != nil {
		if err == sql.ErrNoRows {
			return common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return FormatError(err)
	}
	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	dbSchema, err := raw.toDBSchema()
	if err != nil {
		return err
	}
	return s.cache.UpsertCache(schemaCacheNamespace, upsert.DatabaseID, dbSchema)
}
