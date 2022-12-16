package mongodb

import (
	"context"

	// embed will embeds the migration schema.
	_ "embed"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
)

const (
	// migrationHistoryDefaultDatabase is the default database name for migration history.
	migrationHistoryDefaultDatabase = "bytebase"
	// migrationHistoryDefaultCollection is the default collection name for migration history.
	migrationHistoryDefaultCollection = "migration_history"
)

var (
	//go:embed collmod_migrationHistory_collection_command.json
	collmodMigrationHistoryCollectionCommand string
	//go:embed create_index_on_migrationHistory_collection_command.json
	createIndexOnMigrationHistoryCollectionCommand string
)

// NeedsSetupMigration returns whether the driver needs to setup migration.
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	collectionNames, err := database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list collection names for database %q", migrationHistoryDefaultDatabase)
	}
	for _, collectionName := range collectionNames {
		if collectionName == migrationHistoryDefaultCollection {
			return false, nil
		}
	}
	return true, nil
}

// SetupMigrationIfNeeded sets up migration if needed.
func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return err
	}
	if !setup {
		return nil
	}
	log.Info("Bytebase migration schema not found, creating schema for mongodb...",
		zap.String("environment", driver.connectionCtx.EnvironmentName),
		zap.String("instance", driver.connectionCtx.InstanceName),
	)
	database := driver.client.Database(migrationHistoryDefaultDatabase)
	if err := database.CreateCollection(ctx, migrationHistoryDefaultCollection); err != nil {
		return errors.Wrapf(err, "failed to create collection %q in mongodb database %q for instance named %q", migrationHistoryDefaultCollection, migrationHistoryDefaultDatabase, driver.connectionCtx.InstanceName)
	}

	var b interface{}
	if err := bson.UnmarshalExtJSON([]byte(collmodMigrationHistoryCollectionCommand), true, &b); err != nil {
		return errors.Wrap(err, "failed to unmarshal collmod command")
	}
	var result interface{}
	if err := database.RunCommand(context.Background(), b).Decode(&result); err != nil {
		return errors.Wrap(err, "failed to run collmod command")
	}

	if err := bson.UnmarshalExtJSON([]byte(createIndexOnMigrationHistoryCollectionCommand), true, &b); err != nil {
		return errors.Wrap(err, "failed to unmarshal create index command")
	}
	if err := database.RunCommand(context.Background(), b).Decode(&result); err != nil {
		return errors.Wrap(err, "failed to run create index command")
	}
	log.Info("Successfully created migration schema for mongodb.",
		zap.String("environment", driver.connectionCtx.EnvironmentName),
		zap.String("instance", driver.connectionCtx.InstanceName),
	)
	return nil
}

// ExecuteMigration executes a migration.
func (*Driver) ExecuteMigration(_ context.Context, _ *db.MigrationInfo, _ string) (int64, string, error) {
	panic("not implemented")
}

// FindMigrationHistoryList finds the migration history list.
func (*Driver) FindMigrationHistoryList(_ context.Context, _ *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	// TODO(zp): implement
	return []*db.MigrationHistory{
		{
			Version: "0000.0000.0000-FAKEIMPLEMENT",
		},
	}, nil
}
