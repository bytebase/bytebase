package redis

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Migration related.

// NeedsSetupMigration checks whether we need to setup migration (e.g. creating/upgrading the migration related tables).
// No need because redis uses bytebase metaDB InstanceChangeHistory.
func (*Driver) NeedsSetupMigration(context.Context) (bool, error) {
	return false, nil
}

// SetupMigrationIfNeeded create or upgrade migration related tables.
// No need for redis because it uses bytebase metaDB InstanceChangeHistory.
func (*Driver) SetupMigrationIfNeeded(context.Context) error {
	return nil
}

// ExecuteMigration executes a migration.
// ExecuteMigration will execute the database migration.
// Returns the created migration history id and the updated schema on success.
func (d *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (migrationHistoryID string, updatedSchema string, resErr error) {
	if m.CreateDatabase {
		return "", "", errors.New("redis: creating databases is not supported")
	}
	if _, err := d.Execute(ctx, statement, m.CreateDatabase); err != nil {
		return "", "", util.FormatError(err)
	}
	return "", "", nil
}

// FindMigrationHistoryList finds the migration history list and return most recent item first.
func (*Driver) FindMigrationHistoryList(context.Context, *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	return nil, errors.New("redis: not supported")
}
