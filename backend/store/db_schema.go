package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// UpdateDBSchemaMessage is the message for updating db schema.
type UpdateDBSchemaMessage struct {
	Config *storepb.DatabaseConfig
}

type FindDBSchemaMessage struct {
	InstanceID   string
	DatabaseName string
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, find *FindDBSchemaMessage) (*model.DatabaseMetadata, error) {
	q := qb.Q().Space(`
		SELECT
			metadata,
			raw_dump,
			config
		FROM db_schema
		WHERE instance = ? AND db_name = ?`, find.InstanceID, find.DatabaseName)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var metadata, schema, config []byte
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
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

	dbMetadata, err := s.convertMetadataAndConfig(ctx, metadata, schema, config, find.InstanceID)
	if err != nil {
		return nil, err
	}

	return dbMetadata, nil
}

// UpsertDBSchema upserts a database schema.
func (s *Store) UpsertDBSchema(
	ctx context.Context,
	instanceID,
	databaseName string,
	dbMetadata *storepb.DatabaseSchemaMetadata,
	dbConfig *storepb.DatabaseConfig,
	rawDump []byte,
) error {
	metadataBytes, err := protojson.Marshal(dbMetadata)
	if err != nil {
		return err
	}
	configBytes, err := protojson.Marshal(dbConfig)
	if err != nil {
		return err
	}

	q := qb.Q().Space(`
		INSERT INTO db_schema (
			instance,
			db_name,
			metadata,
			raw_dump,
			config
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(instance, db_name) DO UPDATE SET
			metadata = EXCLUDED.metadata,
			raw_dump = EXCLUDED.raw_dump,
			config = EXCLUDED.config
		RETURNING metadata, raw_dump, config`,
		instanceID,
		databaseName,
		metadataBytes,
		// Convert to string because []byte{} is null which violates db schema constraints.
		string(rawDump),
		configBytes)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var metadata, schema, config []byte
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&metadata,
		&schema,
		&config,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateDBSchema updates a database schema.
func (s *Store) UpdateDBSchema(ctx context.Context, instanceID, databaseName string, patch *UpdateDBSchemaMessage) error {
	set := qb.Q()
	if v := patch.Config; v != nil {
		bytes, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set.Comma("config = ?", bytes)
	}
	if set.Len() == 0 {
		return errors.New("no fields to update")
	}

	q := qb.Q().Space("UPDATE db_schema SET ? WHERE instance = ? AND db_name = ?", set, instanceID, databaseName)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) convertMetadataAndConfig(ctx context.Context, metadata, schema, config []byte, instanceID string) (*model.DatabaseMetadata, error) {
	var databaseSchema storepb.DatabaseSchemaMetadata
	var databaseConfig storepb.DatabaseConfig
	if err := common.ProtojsonUnmarshaler.Unmarshal(metadata, &databaseSchema); err != nil {
		return nil, err
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(config, &databaseConfig); err != nil {
		return nil, err
	}
	instance, err := s.GetInstance(ctx, &FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, err
	}
	return model.NewDatabaseMetadata(&databaseSchema, schema, &databaseConfig, instance.Metadata.GetEngine(), IsObjectCaseSensitive(instance)), nil
}
