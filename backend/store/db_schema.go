package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// UpdateDBSchemaMessage is the message for updating db schema.
type UpdateDBSchemaMessage struct {
	Config *storepb.DatabaseConfig
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, databaseID int) (*model.DBSchema, error) {
	if v, ok := s.dbSchemaCache.Get(databaseID); ok {
		return v, nil
	}

	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, databaseID)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var metadata, schema, config []byte
	if err := tx.QueryRowContext(ctx, `
		SELECT
			metadata,
			raw_dump,
			config
		FROM db_schema
		WHERE `+strings.Join(where, " AND "),
		args...,
	).Scan(
		&metadata,
		&schema,
		&config,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	dbSchema, err := convertMetadataAndConfig(metadata, schema, config)
	if err != nil {
		return nil, err
	}

	s.dbSchemaCache.Add(databaseID, dbSchema)
	return dbSchema, nil
}

// UpsertDBSchema upserts a database schema.
func (s *Store) UpsertDBSchema(ctx context.Context, databaseID int, dbSchema *model.DBSchema, updaterID int) error {
	metadataBytes, err := protojson.Marshal(dbSchema.GetMetadata())
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
		RETURNING metadata, raw_dump, config
	`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var metadata, schema, config []byte
	if err := tx.QueryRowContext(ctx, query,
		updaterID,
		updaterID,
		databaseID,
		metadataBytes,
		// Convert to string because []byte{} is null which violates db schema constraints.
		string(dbSchema.GetSchema()),
	).Scan(
		&metadata,
		&schema,
		&config,
	); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	updatedDBSchema, err := convertMetadataAndConfig(metadata, schema, config)
	if err != nil {
		return err
	}

	s.dbSchemaCache.Add(databaseID, updatedDBSchema)
	return nil
}

// UpdateDBSchema updates a database schema.
func (s *Store) UpdateDBSchema(ctx context.Context, databaseID int, patch *UpdateDBSchemaMessage, updaterID int) error {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", updaterID)}
	if v := patch.Config; v != nil {
		bytes, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("config = $%d", len(args)+1)), append(args, bytes)
	}
	args = append(args, databaseID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE db_schema
			SET `+strings.Join(set, ", ")+`
			WHERE database_id = $%d
		`, len(args)), args...,
	); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Invalid the cache and read the value again.
	s.dbSchemaCache.Remove(databaseID)
	return nil
}

func convertMetadataAndConfig(metadata, schema, config []byte) (*model.DBSchema, error) {
	var databaseSchema storepb.DatabaseSchemaMetadata
	var databaseConfig storepb.DatabaseConfig
	decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := decoder.Unmarshal(metadata, &databaseSchema); err != nil {
		return nil, err
	}
	if err := decoder.Unmarshal(config, &databaseConfig); err != nil {
		return nil, err
	}
	return model.NewDBSchema(&databaseSchema, schema, &databaseConfig), nil
}
