package pg

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	//go:embed pg_migration_schema.sql
	migrationSchema string
)

// NeedsSetupMigration returns whether it needs to setup migration.
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	// Don't use `bytebase` when user gives database instead of instance.
	if !driver.strictUseDb() {
		exist, err := driver.hasBytebaseDatabase(ctx)
		if err != nil {
			return false, err
		}
		if !exist {
			return true, nil
		}
		if _, err := driver.GetDBConnection(ctx, db.BytebaseDatabase); err != nil {
			return false, err
		}
	}

	const query = `
		SELECT
		    1
		FROM information_schema.tables
		WHERE table_name = 'migration_history'
	`

	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		return false, nil
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	return true, nil
}

// SetupMigrationIfNeeded sets up migration if needed.
func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return err
	}

	if setup {
		log.Info("Bytebase migration schema not found, creating schema...",
			zap.String("environment", driver.connectionCtx.EnvironmentID),
			zap.String("instance", driver.connectionCtx.InstanceID),
		)

		// Only try to create `bytebase` db when user provide an instance
		if !driver.strictUseDb() {
			exist, err := driver.hasBytebaseDatabase(ctx)
			if err != nil {
				log.Error("Failed to find database \"bytebase\".",
					zap.Error(err),
					zap.String("environment", driver.connectionCtx.EnvironmentID),
					zap.String("instance", driver.connectionCtx.InstanceID),
				)
				return errors.Wrap(err, "failed to find database \"bytebase\"")
			}

			if !exist {
				// Create `bytebase` database
				if _, err := driver.db.ExecContext(ctx, createBytebaseDatabaseStmt); err != nil {
					log.Error("Failed to create database \"bytebase\".",
						zap.Error(err),
						zap.String("environment", driver.connectionCtx.EnvironmentID),
						zap.String("instance", driver.connectionCtx.InstanceID),
					)
					return util.FormatErrorWithQuery(err, createBytebaseDatabaseStmt)
				}
			}

			if _, err := driver.GetDBConnection(ctx, db.BytebaseDatabase); err != nil {
				log.Error("Failed to switch to database \"bytebase\".",
					zap.Error(err),
					zap.String("environment", driver.connectionCtx.EnvironmentID),
					zap.String("instance", driver.connectionCtx.InstanceID),
				)
				return errors.Wrap(err, "failed to switch to database \"bytebase\"")
			}
		}

		// Create `migration_history` table
		if _, err := driver.db.ExecContext(ctx, migrationSchema); err != nil {
			log.Error("Failed to initialize migration schema.",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentID),
				zap.String("instance", driver.connectionCtx.InstanceID),
			)
			return util.FormatErrorWithQuery(err, migrationSchema)
		}
		log.Info("Successfully created migration schema.",
			zap.String("environment", driver.connectionCtx.EnvironmentID),
			zap.String("instance", driver.connectionCtx.InstanceID),
		)
	}

	return nil
}

// FindLargestVersionSinceBaseline will find the largest version since last baseline or branch.
func (driver *Driver) FindLargestVersionSinceBaseline(ctx context.Context, tx *sql.Tx, namespace string) (*string, error) {
	largestBaselineSequence, err := driver.FindLargestSequence(ctx, tx, namespace, true /* baseline */)
	if err != nil {
		return nil, err
	}
	const getLargestVersionSinceLastBaselineQuery = `
		SELECT MAX(version) FROM migration_history
		WHERE namespace = $1 AND sequence >= $2
	`
	var version sql.NullString
	if err := tx.QueryRowContext(ctx, getLargestVersionSinceLastBaselineQuery,
		namespace, largestBaselineSequence,
	).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, util.FormatErrorWithQuery(err, getLargestVersionSinceLastBaselineQuery)
	}
	if version.Valid {
		return &version.String, nil
	}
	return nil, nil
}

// FindLargestSequence will return the largest sequence number.
func (*Driver) FindLargestSequence(ctx context.Context, tx *sql.Tx, namespace string, baseline bool) (int, error) {
	findLargestSequenceQuery := `
		SELECT MAX(sequence) FROM migration_history
		WHERE namespace = $1`
	if baseline {
		findLargestSequenceQuery = fmt.Sprintf("%s AND (type = '%s' OR type = '%s')", findLargestSequenceQuery, db.Baseline, db.Branch)
	}
	var sequence sql.NullInt32
	if err := tx.QueryRowContext(ctx, findLargestSequenceQuery,
		namespace,
	).Scan(&sequence); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return -1, util.FormatErrorWithQuery(err, findLargestSequenceQuery)
	}
	if sequence.Valid {
		return int(sequence.Int32), nil
	}
	// Returns 0 if we haven't applied any migration for this namespace.
	return 0, nil
}

// InsertPendingHistory will insert the migration record with pending status and return the inserted ID.
func (*Driver) InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (string, error) {
	const insertHistoryQuery = `
	INSERT INTO migration_history (
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		source,
		type,
		status,
		version,
		description,
		statement,
		` + `"schema",` + `
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
	)
	VALUES ($1, EXTRACT(epoch from NOW()), $2, EXTRACT(epoch from NOW()), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 0, $14, $15)
	RETURNING id
	`
	var insertedID string
	if err := tx.QueryRowContext(ctx, insertHistoryQuery,
		m.Creator,
		m.Creator,
		m.ReleaseVersion,
		m.Namespace,
		sequence,
		m.Source,
		m.Type,
		db.Pending,
		storedVersion,
		m.Description,
		statement,
		prevSchema,
		prevSchema,
		m.IssueID,
		m.Payload,
	).Scan(&insertedID); err != nil {
		return "", err
	}
	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (*Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID string) error {
	const updateHistoryAsDoneQuery = `
	UPDATE
		migration_history
	SET
		status = $1,
		execution_duration_ns = $2,
		"schema" = $3
	WHERE id = $4
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, db.Done, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (*Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID string) error {
	const updateHistoryAsFailedQuery = `
	UPDATE
		migration_history
	SET
		status = $1,
		execution_duration_ns = $2
	WHERE id = $3
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsFailedQuery, db.Failed, migrationDurationNs, insertedID)
	return err
}

// ExecuteMigrationUsingMigrationHistory will execute the migration and stores the record to migration history.
func (driver *Driver) ExecuteMigrationUsingMigrationHistory(ctx context.Context, m *db.MigrationInfo, statement string) (string, string, error) {
	if driver.strictUseDb() {
		return driver.executeMigration(ctx, m, statement, driver.strictDatabase)
	}
	return driver.executeMigration(ctx, m, statement, db.BytebaseDatabase)
}

// executeMigration will execute the database migration.
// Returns the created migration history id and the updated schema on success.
func (driver *Driver) executeMigration(ctx context.Context, m *db.MigrationInfo, statement string, databaseName string) (migrationHistoryID string, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't existed yet or is schemaless (e.g. Mongo).
	if !m.CreateDatabase {
		// For baseline migration, we also record the live schema to detect the schema drift.
		// See https://bytebase.com/blog/what-is-database-schema-drift
		if _, err := driver.Dump(ctx, m.Database, &prevSchemaBuf, true /* schemaOnly */); err != nil {
			return "", "", err
		}
	}

	// Phase 1 - Pre-check before executing migration
	// Phase 2 - Record migration history as PENDING
	insertedID, err := driver.beginMigration(ctx, m, prevSchemaBuf.String(), statement, databaseName)
	if err != nil {
		if common.ErrorCode(err) == common.MigrationAlreadyApplied {
			return insertedID, prevSchemaBuf.String(), nil
		}
		return "", "", errors.Wrapf(err, "failed to begin migration for issue %s", m.IssueID)
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := driver.endMigration(ctx, startedNs, insertedID, updatedSchema, databaseName, resErr == nil /* isDone */); err != nil {
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
		// Switch to the target database only if we're NOT creating this target database.
		// We should not call getDBConnection() if the instance is MongoDB because it doesn't support.
		if !m.CreateDatabase {
			if _, err := driver.GetDBConnection(ctx, m.Database); err != nil {
				return "", "", err
			}
		}
		if _, err := driver.Execute(ctx, statement, m.CreateDatabase); err != nil {
			return "", "", err
		}
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, m.Database, &afterSchemaBuf, true /* schemaOnly */); err != nil {
		// We will ignore the dump error if the database is dropped.
		if strings.Contains(err.Error(), "not found") {
			return insertedID, "", nil
		}
		return "", "", err
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// beginMigration checks before executing migration and inserts a migration history record with pending status.
func (driver *Driver) beginMigration(ctx context.Context, m *db.MigrationInfo, prevSchema string, statement string, databaseName string) (insertedID string, err error) {
	// Convert version to stored version.
	storedVersion, err := util.ToStoredVersion(m.UseSemanticVersion, m.Version, m.SemanticVersionSuffix)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert to stored version")
	}
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
		Database: &m.Namespace,
		Version:  &m.Version,
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

	// We use transaction here for the RDBMS.
	sqldb, err := driver.GetDBConnection(ctx, databaseName)
	if err != nil {
		return "", err
	}
	// From a concurrency perspective, there's no difference between using transaction or not. However, we use transaction here to save some code of starting a transaction inside each db engine executor.
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	largestSequence, err := driver.FindLargestSequence(ctx, tx, m.Namespace, false /* baseline */)
	if err != nil {
		return "", err
	}

	// Check if there is any higher version already been applied since the last baseline or branch.
	if version, err := driver.FindLargestVersionSinceBaseline(ctx, tx, m.Namespace); err != nil {
		return "", err
	} else if version != nil && len(*version) > 0 && *version >= m.Version {
		// len(*version) > 0 is used because Clickhouse will always return non-nil version with empty string.
		return "", common.Errorf(common.MigrationOutOfOrder, "database %q has already applied version %s which >= %s", m.Database, *version, m.Version)
	}

	// Phase 2 - Record migration history as PENDING.
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	if insertedID, err = driver.InsertPendingHistory(ctx, tx, largestSequence+1, prevSchema, m, storedVersion, statementRecord); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return insertedID, nil
}

// endMigration updates the migration history record to DONE or FAILED depending on migration is done or not.
func (driver *Driver) endMigration(ctx context.Context, startedNs int64, migrationHistoryID string, updatedSchema string, databaseName string, isDone bool) (err error) {
	migrationDurationNs := time.Now().UnixNano() - startedNs

	sqldb, err := driver.GetDBConnection(ctx, databaseName)
	if err != nil {
		return err
	}
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if isDone {
		// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
		err = driver.UpdateHistoryAsDone(ctx, tx, migrationDurationNs, updatedSchema, migrationHistoryID)
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		err = driver.UpdateHistoryAsFailed(ctx, tx, migrationDurationNs, migrationHistoryID)
	}

	if err != nil {
		return err
	}
	return tx.Commit()
}

// FindMigrationHistoryList finds the migration history.
func (driver *Driver) FindMigrationHistoryList(ctx context.Context, find *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	baseQuery := `
	SELECT
		id,
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		source,
		type,
		status,
		version,
		description,
		statement,
		` + `"schema",` + `
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM migration_history `
	paramNames, params := []string{}, []any{}
	if v := find.ID; v != nil {
		paramNames, params = append(paramNames, "id"), append(params, *v)
	}
	if v := find.Database; v != nil {
		paramNames, params = append(paramNames, "namespace"), append(params, *v)
	}
	if v := find.Version; v != nil {
		// TODO(d): support semantic versioning.
		storedVersion, err := util.ToStoredVersion(false, *v, "")
		if err != nil {
			return nil, err
		}
		paramNames, params = append(paramNames, "version"), append(params, storedVersion)
	}
	if v := find.Source; v != nil {
		paramNames, params = append(paramNames, "source"), append(params, *v)
	}
	var query = baseQuery +
		db.FormatParamNameInNumberedPosition(paramNames) +
		`ORDER BY id DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		query += fmt.Sprintf(" OFFSET %d", *v)
	}

	database := db.BytebaseDatabase
	if driver.strictUseDb() {
		database = driver.strictDatabase
	}
	db, err := driver.GetDBConnection(ctx, database)
	if err != nil {
		return nil, err
	}
	return util.FindMigrationHistoryList(ctx, query, params, db)
}

func (driver *Driver) hasBytebaseDatabase(ctx context.Context) (bool, error) {
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return false, err
	}
	for _, database := range databases {
		if database.Name == db.BytebaseDatabase {
			return true, nil
		}
	}
	return false, nil
}

func (driver *Driver) strictUseDb() bool {
	return len(driver.strictDatabase) != 0
}
