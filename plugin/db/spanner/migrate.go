package spanner

import (
	"context"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NeedsSetupMigration checks if it needs to set up migration.
func (d *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	_, err := d.dbClient.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: "bytebase"})
	if status.Code(err) == codes.NotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

// SetupMigrationIfNeeded sets up migration if needed.
func (d *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := d.NeedsSetupMigration(ctx)
	if err != nil {
		return err
	}
	if !setup {
		return nil
	}
	log.Info("Bytebase migration schema not found, creating schema...",
		zap.String("environment", d.connCtx.EnvironmentName),
		zap.String("instance", d.connCtx.InstanceName),
	)
	statements := splitStatement(migrationSchema)
	op, err := d.dbClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          d.config.Host,
		CreateStatement: "CREATE DATABASE bytebase",
		ExtraStatements: statements,
	})
	if err != nil {
		return err
	}
	if _, err := op.Wait(ctx); err != nil {
		return err
	}
	return nil
}

// ExecuteMigration executes a migration.
func (*Driver) ExecuteMigration(_ context.Context, _ *db.MigrationInfo, _ string) (int64, string, error) {
	panic("not implemented")
}

// FindMigrationHistoryList finds the migration history list.
func (*Driver) FindMigrationHistoryList(_ context.Context, _ *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	panic("not implemented")
}
