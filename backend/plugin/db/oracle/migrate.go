package snowflake

import (
	"context"
	"database/sql"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ util.MigrationExecutor = (*Driver)(nil)
)

// NeedsSetupMigration returns whether it needs to setup migration.
func (*Driver) NeedsSetupMigration(_ context.Context) (bool, error) {
	// TODO(d): implement it.
	return false, nil
}

// SetupMigrationIfNeeded sets up migration if needed.
func (*Driver) SetupMigrationIfNeeded(_ context.Context) error {
	// TODO(d): implement it.
	return nil
}

// FindLargestVersionSinceBaseline will find the largest version since last baseline or branch.
func (Driver) FindLargestVersionSinceBaseline(_ context.Context, _ *sql.Tx, _ string) (*string, error) {
	// TODO(d): implement it.
	return nil, nil
}

// FindLargestSequence will return the largest sequence number.
func (Driver) FindLargestSequence(_ context.Context, _ *sql.Tx, _ string, _ bool) (int, error) {
	// TODO(d): implement it.
	return 0, nil
}

// InsertPendingHistory will insert the migration record with pending status and return the inserted ID.
func (Driver) InsertPendingHistory(_ context.Context, _ *sql.Tx, _ int, _ string, _ *db.MigrationInfo, _, _ string) (string, error) {
	// TODO(d): implement it.
	return "", nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(_ context.Context, _ *sql.Tx, _ int64, _ string, _ string) error {
	// TODO(d): implement it.
	return nil
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(_ context.Context, _ *sql.Tx, _ int64, _ string) error {
	// TODO(d): implement it.
	return nil
}

// ExecuteMigration will execute the migration.
func (*Driver) ExecuteMigration(_ context.Context, _ *db.MigrationInfo, _ string) (string, string, error) {
	// TODO(d): implement it.
	return "", "", nil
}

// FindMigrationHistoryList finds the migration history.
func (*Driver) FindMigrationHistoryList(_ context.Context, _ *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	// TODO(d): implement it.
	return nil, nil
}
