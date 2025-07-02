package migrator

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	migrationInterval = 10 * time.Second
	batchSize         = 20
)

// ColumnDefaultMigrator is the migrator for column default values.
// It will migrate the `default_null` and `default_expression` to `default`.
// The process is asynchronous and will not block the application startup.
// The migrator will be started after the server is started.
type ColumnDefaultMigrator struct {
	store            *store.Store
	supportedEngines []storepb.Engine
}

// NewColumnDefaultMigrator creates a new ColumnDefaultMigrator.
func NewColumnDefaultMigrator(store *store.Store, supportedEngines []storepb.Engine) *ColumnDefaultMigrator {
	return &ColumnDefaultMigrator{
		store:            store,
		supportedEngines: supportedEngines,
	}
}

// EnginesNeedingMigration returns the list of engines that currently need column default migration.
// This function dynamically builds the list based on common.EngineNeedsColumnDefaultMigration.
func EnginesNeedingMigration() []storepb.Engine {
	var engines []storepb.Engine
	// Check all known engines
	for _, engine := range storepb.Engine_value {
		if common.EngineDBSchemaReadyToMigrate(storepb.Engine(engine)) {
			engines = append(engines, storepb.Engine(engine))
		}
	}
	return engines
}

// Run starts the ColumnDefaultMigrator.
func (m *ColumnDefaultMigrator) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(migrationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := m.migrate(ctx); err != nil {
				slog.Error("Failed to run column default value migrator", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *ColumnDefaultMigrator) migrate(ctx context.Context) error {
	for _, engine := range m.supportedEngines {
		dbSchemas, err := m.store.ListDBSchemasWithTodo(ctx, engine, batchSize)
		if err != nil {
			return err
		}

		if len(dbSchemas) == 0 {
			continue
		}

		for _, dbSchema := range dbSchemas {
			metadata := &storepb.DatabaseSchemaMetadata{}
			if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(dbSchema.Metadata), metadata); err != nil {
				slog.Warn("Failed to unmarshal db schema metadata", slog.Int("db_schema_id", dbSchema.ID), slog.String("db_name", dbSchema.DBName), log.BBError(err))
				// We mark the row as done even if it fails to unmarshal, to avoid retrying forever.
				if err := m.store.UpdateDBSchemaTodo(ctx, dbSchema.ID, false); err != nil {
					slog.Error("Failed to mark db schema as done", slog.Int("db_schema_id", dbSchema.ID), slog.String("db_name", dbSchema.DBName), log.BBError(err))
				}
				continue
			}

			changed := false
			for _, schema := range metadata.Schemas {
				for _, table := range schema.Tables {
					for _, column := range table.Columns {
						if column.DefaultNull {
							column.Default = "NULL"
							column.DefaultNull = false
							changed = true
						} else if column.DefaultExpression != "" {
							column.Default = column.DefaultExpression
							column.DefaultExpression = ""
							changed = true
						}
					}
				}
			}

			// If no changes were made, just mark as done.
			if !changed {
				if err := m.store.UpdateDBSchemaTodo(ctx, dbSchema.ID, false); err != nil {
					slog.Error("Failed to mark db schema as done", slog.Int("db_schema_id", dbSchema.ID), slog.String("db_name", dbSchema.DBName), log.BBError(err))
				}
				continue
			}

			marshaled, err := protojson.Marshal(metadata)
			if err != nil {
				slog.Error("Failed to marshal db schema metadata", slog.Int("db_schema_id", dbSchema.ID), slog.String("db_name", dbSchema.DBName), log.BBError(err))
				continue
			}

			// Update metadata and todo in a single transaction, only if todo is still true.
			// This prevents race conditions with the sync process.
			if err := m.store.UpdateDBSchemaMetadataIfTodo(ctx, dbSchema.ID, string(marshaled)); err != nil {
				slog.Error("Failed to update db schema metadata and todo", slog.Int("db_schema_id", dbSchema.ID), slog.String("db_name", dbSchema.DBName), log.BBError(err))
				continue
			}
		}
	}

	return nil
}
