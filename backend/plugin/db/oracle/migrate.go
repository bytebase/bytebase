package oracle

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Migration related.

// NeedsSetupMigration checks whether we need to setup migration (e.g. creating/upgrading the migration related tables).
func (*Driver) NeedsSetupMigration(context.Context) (bool, error) {
	return false, nil
}

// SetupMigrationIfNeeded create or upgrade migration related tables.
func (*Driver) SetupMigrationIfNeeded(context.Context) error {
	return nil
}

// ExecuteMigration executes a migration.
// ExecuteMigration will execute the database migration.
// Returns the created migration history id and the updated schema on success.
func (d *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (migrationHistoryID string, updatedSchema string, resErr error) {
	return d.ExecuteMigrationWithBeforeCommitTxFunc(ctx, m, statement, nil)
}

// ExecuteMigrationWithBeforeCommitTxFunc executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
//
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (d *Driver) ExecuteMigrationWithBeforeCommitTxFunc(ctx context.Context, m *db.MigrationInfo, statement string, beforeCommitTxFunc func(tx *sql.Tx) error) (migrationHistoryID string, updatedSchema string, resErr error) {
	if m.CreateDatabase {
		return "", "", errors.New("creating databases is not supported")
	}
	if _, err := d.executeWithBeforeCommitTxFunc(ctx, statement, m.CreateDatabase, beforeCommitTxFunc); err != nil {
		return "", "", util.FormatError(err)
	}
	return "", "", nil
}

// FindMigrationHistoryList finds the migration history list and return most recent item first.
func (*Driver) FindMigrationHistoryList(context.Context, *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	return nil, errors.New("unsupported")
}
