package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DBSchema is the database schema including the metadata and schema (raw dump).
type DBSchema struct {
	Metadata *storepb.DatabaseMetadata
	Schema   []byte
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, databaseID int) (*DBSchema, error) {
	if dbSchema, ok := s.dbSchemaCache[databaseID]; ok {
		return dbSchema, nil
	}

	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, databaseID)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	dbSchema := &DBSchema{}
	var metadata []byte
	if err := tx.QueryRowContext(ctx, `
		SELECT
			metadata,
			raw_dump
		FROM db_schema
		WHERE `+strings.Join(where, " AND "),
		args...,
	).Scan(
		&metadata,
		&dbSchema.Schema,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	var databaseSchema storepb.DatabaseMetadata
	decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := decoder.Unmarshal(metadata, &databaseSchema); err != nil {
		return nil, err
	}
	dbSchema.Metadata = &databaseSchema

	s.dbSchemaCache[databaseID] = dbSchema
	return dbSchema, nil
}

// UpsertDBSchema upserts a database schema.
func (s *Store) UpsertDBSchema(ctx context.Context, databaseID int, dbSchema *DBSchema, updaterID int) error {
	metadataBytes, err := protojson.Marshal(dbSchema.Metadata)
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

	if _, err := tx.ExecContext(ctx, query,
		updaterID,
		updaterID,
		databaseID,
		metadataBytes,
		// Convert to string because []byte{} is null which violates db schema constraints.
		string(dbSchema.Schema),
	); err != nil {
		return FormatError(err)
	}
	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	s.dbSchemaCache[databaseID] = dbSchema
	return nil
}
