package store

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DBSchema is the database schema including the metadata and schema (raw dump).
type DBSchema struct {
	Metadata *storepb.DatabaseSchemaMetadata
	Schema   []byte
}

// TableExists checks if the table exists.
func (s *DBSchema) TableExists(schemaName string, tableName string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		schemaName = strings.ToLower(schemaName)
		tableName = strings.ToLower(tableName)
	}
	for _, schema := range s.Metadata.Schemas {
		currentSchemaName := schema.Name
		if ignoreCaseSensitive {
			currentSchemaName = strings.ToLower(currentSchemaName)
		}
		if currentSchemaName != schemaName {
			continue
		}
		for _, table := range schema.Tables {
			currentTableName := table.Name
			if ignoreCaseSensitive {
				currentTableName = strings.ToLower(currentTableName)
			}
			if currentTableName == tableName {
				return true
			}
		}
	}
	return false
}

// ViewExists checks if the view exists.
func (s *DBSchema) ViewExists(schemaName string, name string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		schemaName = strings.ToLower(schemaName)
		name = strings.ToLower(name)
	}
	for _, schema := range s.Metadata.Schemas {
		currentSchemaName := schema.Name
		if ignoreCaseSensitive {
			currentSchemaName = strings.ToLower(currentSchemaName)
		}
		if currentSchemaName != schemaName {
			continue
		}
		for _, view := range schema.Views {
			currentViewName := view.Name
			if ignoreCaseSensitive {
				currentViewName = strings.ToLower(currentViewName)
			}
			if currentViewName == name {
				return true
			}
		}
	}
	return false
}

// CompactText returns the compact text representation of the database schema.
func (s *DBSchema) CompactText() (string, error) {
	if s.Metadata == nil {
		return "", nil
	}

	var buf bytes.Buffer
	for _, schema := range s.Metadata.Schemas {
		schemaName := schema.Name
		// If the schema name is empty, use the database name instead, such as MySQL.
		if schemaName == "" {
			schemaName = s.Metadata.Name
		}
		for _, table := range schema.Tables {
			// Table with columns.
			if _, err := buf.WriteString(fmt.Sprintf("# Table %s.%s(", schemaName, table.Name)); err != nil {
				return "", err
			}
			for i, column := range table.Columns {
				if i == 0 {
					if _, err := buf.WriteString(column.Name); err != nil {
						return "", err
					}
				} else {
					if _, err := buf.WriteString(fmt.Sprintf(", %s", column.Name)); err != nil {
						return "", err
					}
				}
			}
			if _, err := buf.WriteString(") #\n"); err != nil {
				return "", err
			}

			// Indexes.
			for _, index := range table.Indexes {
				if _, err := buf.WriteString(fmt.Sprintf("# Index %s(%s) ON table %s.%s #\n", index.Name, strings.Join(index.Expressions, ", "), schemaName, table.Name)); err != nil {
					return "", err
				}
			}
		}
	}

	return buf.String(), nil
}

// FindIndex finds the index by name.
func (s *DBSchema) FindIndex(schemaName string, tableName string, indexName string) *storepb.IndexMetadata {
	for _, schema := range s.Metadata.Schemas {
		if schema.Name != schemaName {
			continue
		}
		for _, table := range schema.Tables {
			if table.Name != tableName {
				continue
			}
			for _, index := range table.Indexes {
				if index.Name == indexName {
					return index
				}
			}
		}
	}
	return nil
}

// GetDBSchema gets the schema for a database.
func (s *Store) GetDBSchema(ctx context.Context, databaseID int) (*DBSchema, error) {
	if dbSchema, ok := s.dbSchemaCache.Load(databaseID); ok {
		return dbSchema.(*DBSchema), nil
	}

	// Build WHERE clause.
	where, args := []string{"TRUE"}, []any{}
	where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, databaseID)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var databaseSchema storepb.DatabaseSchemaMetadata
	decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := decoder.Unmarshal(metadata, &databaseSchema); err != nil {
		return nil, err
	}
	dbSchema.Metadata = &databaseSchema

	s.dbSchemaCache.Store(databaseID, dbSchema)
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
		return err
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
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	s.dbSchemaCache.Store(databaseID, dbSchema)
	return nil
}
