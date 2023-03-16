package pg

import (
	"context"
	"database/sql"
	"fmt"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	//go:embed pg_migration_schema.sql
	migrationSchema string

	_ util.MigrationExecutor = (*Driver)(nil)
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
		if err := driver.switchDatabase(db.BytebaseDatabase); err != nil {
			return false, err
		}
	}

	const query = `
		SELECT
		    1
		FROM information_schema.tables
		WHERE table_name = 'migration_history'
	`

	return util.NeedsSetupMigrationSchema(ctx, driver.db, query)
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

			if err := driver.switchDatabase(db.BytebaseDatabase); err != nil {
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

// ExecuteMigration will execute the migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (string, string, error) {
	_, err := driver.Execute(ctx, statement, m.CreateDatabase)
	return "", "", err
}

// ExecuteMigrationUsingMigrationHistory will execute the migration and stores the record to migration history.
func (driver *Driver) ExecuteMigrationUsingMigrationHistory(ctx context.Context, m *db.MigrationInfo, statement string) (string, string, error) {
	if driver.strictUseDb() {
		return util.ExecuteMigration(ctx, driver, m, statement, driver.strictDatabase)
	}
	return util.ExecuteMigration(ctx, driver, m, statement, db.BytebaseDatabase)
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
	return util.FindMigrationHistoryList(ctx, query, params, driver, database)
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
