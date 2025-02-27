package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// UpdateDBSchemaMessage is the message for updating db schema.
type UpdateDBSchemaMessage struct {
	Config *storepb.DatabaseConfig
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, instanceID, databaseName string) (*model.DatabaseSchema, error) {
	if v, ok := s.dbSchemaCache.Get(getDatabaseCacheKey(instanceID, databaseName)); ok {
		return v, nil
	}

	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("instance = $%d", len(args)+1)), append(args, instanceID)
	where, args = append(where, fmt.Sprintf("db_name = $%d", len(args)+1)), append(args, databaseName)

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

	dbSchema, err := s.convertMetadataAndConfig(ctx, metadata, schema, config, instanceID)
	if err != nil {
		return nil, err
	}

	s.dbSchemaCache.Add(getDatabaseCacheKey(instanceID, databaseName), dbSchema)
	return dbSchema, nil
}

// UpsertDBSchema upserts a database schema.
func (s *Store) UpsertDBSchema(
	ctx context.Context,
	instanceID,
	databaseName string,
	dbMetadata *storepb.DatabaseSchemaMetadata,
	dbConfig *storepb.DatabaseConfig,
	dbSchema []byte,
) error {
	metadataBytes, err := protojson.Marshal(dbMetadata)
	if err != nil {
		return err
	}
	configBytes, err := protojson.Marshal(dbConfig)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO db_schema (
			instance,
			db_name,
			metadata,
			raw_dump,
			config
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(instance, db_name) DO UPDATE SET
			metadata = EXCLUDED.metadata,
			raw_dump = EXCLUDED.raw_dump,
			config = EXCLUDED.config
		RETURNING metadata, raw_dump, config
	`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var metadata, schema, config []byte
	if err := tx.QueryRowContext(ctx, query,
		instanceID,
		databaseName,
		metadataBytes,
		// Convert to string because []byte{} is null which violates db schema constraints.
		string(dbSchema),
		configBytes,
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
	updatedDBSchema, err := s.convertMetadataAndConfig(ctx, metadata, schema, config, instanceID)
	if err != nil {
		return err
	}

	s.dbSchemaCache.Add(getDatabaseCacheKey(instanceID, databaseName), updatedDBSchema)
	return nil
}

// UpdateDBSchema updates a database schema.
func (s *Store) UpdateDBSchema(ctx context.Context, instanceID, databaseName string, patch *UpdateDBSchemaMessage) error {
	set, args := []string{}, []any{}
	if v := patch.Config; v != nil {
		bytes, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("config = $%d", len(args)+1)), append(args, bytes)
	}

	where := []string{"TRUE"}
	where, args = append(where, fmt.Sprintf("instance = $%d", len(args)+1)), append(args, instanceID)
	where, args = append(where, fmt.Sprintf("db_name = $%d", len(args)+1)), append(args, databaseName)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
			UPDATE db_schema
			SET `+strings.Join(set, ", ")+`
			WHERE `+strings.Join(where, " AND "), args...,
	); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Invalid the cache and read the value again.
	s.dbSchemaCache.Remove(getDatabaseCacheKey(instanceID, databaseName))
	return nil
}

func (s *Store) convertMetadataAndConfig(ctx context.Context, metadata, schema, config []byte, instanceID string) (*model.DatabaseSchema, error) {
	var databaseSchema storepb.DatabaseSchemaMetadata
	var databaseConfig storepb.DatabaseConfig
	if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, &databaseSchema); err != nil {
		return nil, err
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(config, &databaseConfig); err != nil {
		return nil, err
	}
	instance, err := s.GetInstanceV2(ctx, &FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, err
	}
	return model.NewDatabaseSchema(&databaseSchema, schema, &databaseConfig, instance.Engine, IsObjectCaseSensitive(instance)), nil
}
