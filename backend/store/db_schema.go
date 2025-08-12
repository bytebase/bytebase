package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// UpdateDBSchemaMessage is the message for updating db schema.
type UpdateDBSchemaMessage struct {
	Config *storepb.DatabaseConfig
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, instanceID, databaseName string) (*model.DatabaseSchema, error) {
	if v, ok := s.dbSchemaCache.Get(getDatabaseCacheKey(instanceID, databaseName)); ok && s.enableCache {
		return v, nil
	}

	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("instance = $%d", len(args)+1)), append(args, instanceID)
	where, args = append(where, fmt.Sprintf("db_name = $%d", len(args)+1)), append(args, databaseName)

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
	todo bool,
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
			config,
			todo
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(instance, db_name) DO UPDATE SET
			metadata = EXCLUDED.metadata,
			raw_dump = EXCLUDED.raw_dump,
			config = EXCLUDED.config,
			todo = EXCLUDED.todo
		RETURNING metadata, raw_dump, config
	`
	tx, err := s.GetDB().BeginTx(ctx, nil)
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
		todo,
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

	tx, err := s.GetDB().BeginTx(ctx, nil)
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

// DBSchemaWithTodo is a struct for a db_schema with the todo column.
// It is used for the column default value migration.
type DBSchemaWithTodo struct {
	ID         int
	InstanceID string
	DBName     string
	Metadata   string
}

// ListDBSchemasWithTodo lists all db_schemas with todo = true.
func (s *Store) ListDBSchemasWithTodo(ctx context.Context, engineType storepb.Engine, limit int) ([]*DBSchemaWithTodo, error) {
	rows, err := s.GetDB().QueryContext(ctx, `
		SELECT
			db_schema.id,
			db_schema.instance,
			db_schema.db_name,
			db_schema.metadata
		FROM db_schema
		LEFT JOIN instance ON db_schema.instance = instance.resource_id
		WHERE db_schema.todo = true AND instance.metadata->>'engine' = $1
		LIMIT $2
	`, engineType.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbSchemas []*DBSchemaWithTodo
	for rows.Next() {
		var dbSchema DBSchemaWithTodo
		if err := rows.Scan(
			&dbSchema.ID,
			&dbSchema.InstanceID,
			&dbSchema.DBName,
			&dbSchema.Metadata,
		); err != nil {
			return nil, err
		}
		dbSchemas = append(dbSchemas, &dbSchema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dbSchemas, nil
}

// UpdateDBSchemaTodo updates the todo column of a db_schema.
func (s *Store) UpdateDBSchemaTodo(ctx context.Context, id int, todo bool) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE db_schema
		SET todo = $1
		WHERE id = $2
	`, todo, id); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateDBSchemaMetadata updates the metadata of a db_schema.
func (s *Store) UpdateDBSchemaMetadata(ctx context.Context, id int, metadata string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE db_schema
		SET metadata = $1
		WHERE id = $2
	`, metadata, id); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateDBSchemaMetadataIfTodo updates the metadata and sets todo to false only if todo is currently true.
// This is used by the migrator to avoid race conditions with the sync process.
// The WHERE condition ensures atomic check-and-update, preventing any race conditions.
func (s *Store) UpdateDBSchemaMetadataIfTodo(ctx context.Context, id int, metadata string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE db_schema
		SET metadata = $1, todo = false
		WHERE id = $2 AND todo = true
	`, metadata, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		// Schema was already processed by sync, skip update
		return tx.Rollback()
	}

	return tx.Commit()
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
	return model.NewDatabaseSchema(&databaseSchema, schema, &databaseConfig, instance.Metadata.GetEngine(), IsObjectCaseSensitive(instance)), nil
}
