package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
)

var (
	//go:embed sqlite_migration_schema.sql
	migrationSchema string

	_ util.MigrationExecutor = (*Driver)(nil)
)

// NeedsSetupMigration returns whether it needs to setup migration.
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	exist, err := driver.hasBytebaseDatabase()
	if err != nil {
		return false, err
	}
	if !exist {
		return true, nil
	}
	if _, err := driver.GetDbConnection(ctx, bytebaseDatabase); err != nil {
		return false, err
	}

	const query = `
		SELECT
		    1
		FROM sqlite_master
		WHERE type='table' AND name = 'bytebase_migration_history'
	`
	return util.NeedsSetupMigrationSchema(ctx, driver.db, query)
}

// SetupMigrationIfNeeded sets up migration if needed.
func (driver *Driver) SetupMigrationIfNeeded(ctx context.Context) error {
	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return nil
	}

	if setup {
		log.Info("Bytebase migration schema not found, creating schema...",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)

		if _, err := driver.GetDbConnection(ctx, bytebaseDatabase); err != nil {
			log.Error("Failed to switch to database \"bytebase\".",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentName),
				zap.String("database", driver.connectionCtx.InstanceName),
			)
			return fmt.Errorf("failed to switch to database \"bytebase\", error: %v", err)
		}

		if _, err := driver.db.ExecContext(ctx, migrationSchema); err != nil {
			log.Error("Failed to initialize migration schema.",
				zap.Error(err),
				zap.String("environment", driver.connectionCtx.EnvironmentName),
				zap.String("database", driver.connectionCtx.InstanceName),
			)
			return util.FormatErrorWithQuery(err, migrationSchema)
		}
		log.Info("Successfully created migration schema.",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)
	}

	return nil
}

// FindLargestVersionSinceBaseline will find the largest version since last baseline or branch.
func (driver Driver) FindLargestVersionSinceBaseline(ctx context.Context, tx *sql.Tx, namespace string) (*string, error) {
	largestBaselineSequence, err := driver.FindLargestSequence(ctx, tx, namespace, true /* baseline */)
	if err != nil {
		return nil, err
	}
	const getLargestVersionSinceLastBaselineQuery = `
		SELECT MAX(version) FROM bytebase_migration_history
		WHERE namespace = ? AND sequence >= ?
	`
	row, err := tx.QueryContext(ctx, getLargestVersionSinceLastBaselineQuery,
		namespace, largestBaselineSequence,
	)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, getLargestVersionSinceLastBaselineQuery)
	}
	defer row.Close()

	var version sql.NullString
	row.Next()
	if err := row.Scan(&version); err != nil {
		return nil, err
	}

	if version.Valid {
		return &version.String, nil
	}

	return nil, nil
}

// FindLargestSequence will return the largest sequence number.
func (Driver) FindLargestSequence(ctx context.Context, tx *sql.Tx, namespace string, baseline bool) (int, error) {
	findLargestSequenceQuery := `
		SELECT MAX(sequence) FROM bytebase_migration_history
		WHERE namespace = ?`
	if baseline {
		findLargestSequenceQuery = fmt.Sprintf("%s AND (type = '%s' OR type = '%s')", findLargestSequenceQuery, db.Baseline, db.Branch)
	}
	row, err := tx.QueryContext(ctx, findLargestSequenceQuery,
		namespace,
	)
	if err != nil {
		return -1, util.FormatErrorWithQuery(err, findLargestSequenceQuery)
	}
	defer row.Close()

	var sequence sql.NullInt32
	row.Next()
	if err := row.Scan(&sequence); err != nil {
		return -1, err
	}

	if !sequence.Valid {
		// Returns 0 if we haven't applied any migration for this namespace.
		return 0, nil
	}

	return int(sequence.Int32), nil
}

// InsertPendingHistory will insert the migration record with pending status and return the inserted ID.
func (Driver) InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (int64, error) {
	const insertHistoryQuery = `
	INSERT INTO bytebase_migration_history (
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
		schema,
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
	)
	VALUES (?, strftime('%s', 'now'), ?, strftime('%s', 'now'), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
	`
	res, err := tx.ExecContext(ctx, insertHistoryQuery,
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
	)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}

	insertedID, err := res.LastInsertId()
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}
	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error {
	const updateHistoryAsDoneQuery = `
	UPDATE
		bytebase_migration_history
	SET
		status = ?,
		execution_duration_ns = ?,
		schema = ?
	WHERE id = ?
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, db.Done, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error {
	const updateHistoryAsFailedQuery = `
	UPDATE
		bytebase_migration_history
	SET
		status = ?,
		execution_duration_ns = ?
	WHERE id = ?
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsFailedQuery, db.Failed, migrationDurationNs, insertedID)
	return err
}

// ExecuteMigration will execute the migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
	return util.ExecuteMigration(ctx, driver, m, statement, bytebaseDatabase)
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
		schema,
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM bytebase_migration_history `
	paramNames, params := []string{}, []interface{}{}
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
		db.FormatParamNameInQuestionMark(paramNames) +
		`ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	return util.FindMigrationHistoryList(ctx, query, params, driver, bytebaseDatabase, find, baseQuery)
}
