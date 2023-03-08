package redis

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
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
func (d *Driver) ExecuteMigration(ctx context.Context, store db.InstanceChangeHistoryStore, m *db.MigrationInfo, statement string) (migrationHistoryID string, updatedSchema string, resErr error) {
	if m.CreateDatabase {
		return "", "", errors.New("redis: creating databases is not supported")
	}
	prevSchema := ""

	// Phase 1 - Pre-check before executing migration
	// Phase 2 - Record migration history as PENDING
	insertedID, err := d.beginMigration(ctx, store, m, prevSchema, statement)
	if err != nil {
		if common.ErrorCode(err) == common.MigrationAlreadyApplied {
			return insertedID, prevSchema, nil
		}
		return "", "", errors.Wrapf(err, "failed to begin migration for issue %s", m.IssueID)
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := d.endMigration(ctx, store, startedNs, insertedID, updatedSchema, db.BytebaseDatabase, resErr == nil /*isDone*/); err != nil {
			log.Error("Failed to update migration history record",
				zap.Error(err),
				zap.String("migration_id", migrationHistoryID),
			)
		}
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute.
	// https://github.com/bytebase/bytebase/issues/394
	doMigrate := true
	if statement == "" {
		doMigrate = false
	}
	if m.Type == db.Baseline {
		doMigrate = false
	}
	if doMigrate {
		if _, err := d.Execute(ctx, statement, m.CreateDatabase); err != nil {
			return "", "", util.FormatError(err)
		}
	}

	// NO phase 4 for redis!
	// Phase 4 - Dump the schema after migration
	return insertedID, "", nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func (*Driver) beginMigration(ctx context.Context, store db.InstanceChangeHistoryStore, m *db.MigrationInfo, prevSchema string, statement string) (string, error) {
	// Convert version to stored version.
	storedVersion, err := util.ToStoredVersion(m.UseSemanticVersion, m.Version, m.SemanticVersionSuffix)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert to stored version")
	}
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := store.FindInstanceChangeHistoryList(ctx, &db.MigrationHistoryFind{
		InstanceID: m.InstanceID,
		DatabaseID: m.DatabaseID,
		Version:    &m.Version,
	}); err != nil {
		return "", errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case db.Done:
			if migrationHistory.IssueID != m.IssueID {
				return migrationHistory.ID, common.Errorf(common.MigrationFailed, "database %q has already applied version %s by issue %s", m.Database, m.Version, migrationHistory.IssueID)
			}
			return migrationHistory.ID, common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s", m.Database, m.Version)
		case db.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.ID, nil
			}
			return "", common.Wrap(err, common.MigrationPending)
		case db.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version ", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return migrationHistory.ID, nil
			}
			return "", common.Wrap(err, common.MigrationFailed)
		}
	}

	largestSequence, err := store.GetLargestInstanceChangeHistorySequence(ctx, m.InstanceID, m.DatabaseID, false /* baseline */)
	if err != nil {
		return "", err
	}

	// Check if there is any higher version already been applied since the last baseline or branch.
	if version, err := store.GetLargestInstanceChangeHistoryVersionSinceBaseline(ctx, m.InstanceID, m.DatabaseID); err != nil {
		return "", err
	} else if version != nil && len(*version) > 0 && *version >= m.Version {
		return "", common.Errorf(common.MigrationOutOfOrder, "database %q has already applied version %s which >= %s", m.Database, *version, m.Version)
	}

	// Phase 2 - Record migration history as PENDING.
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	insertedID, err := store.CreatePendingInstanceChangeHistory(ctx, largestSequence+1, prevSchema, m, storedVersion, statementRecord)
	if err != nil {
		return "", err
	}

	return insertedID, nil
}

// EndMigration updates the migration history record to DONE or FAILED depending on migration is done or not.

func (*Driver) endMigration(ctx context.Context, store db.InstanceChangeHistoryStore, startedNs int64, insertedID string, updatedSchema string, _ string, isDone bool) error {
	var err error
	migrationDurationNs := time.Now().UnixNano() - startedNs

	if isDone {
		err = store.UpdateInstanceChangeHistoryAsDone(ctx, migrationDurationNs, updatedSchema, insertedID)
		// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		err = store.UpdateInstanceChangeHistoryAsFailed(ctx, migrationDurationNs, insertedID)
	}

	return err
}

// FindMigrationHistoryList finds the migration history list and return most recent item first.
func (*Driver) FindMigrationHistoryList(context.Context, *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	return nil, errors.New("redis: not supported")
}
