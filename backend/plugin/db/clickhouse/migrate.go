package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// embed will embeds the migration schema.
	_ "embed"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	//go:embed clickhouse_migration_schema.sql
	migrationSchema string

	_ util.MigrationExecutor = (*Driver)(nil)
)

// NeedsSetupMigration returns whether it needs to setup migration.
func (driver *Driver) NeedsSetupMigration(ctx context.Context) (bool, error) {
	const query = `
		SELECT
			1
		FROM system.tables
		WHERE database = 'bytebase' AND name = 'migration_history'
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
		if _, err := driver.Execute(ctx, migrationSchema, true /* createDatabase */); err != nil {
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
func (driver Driver) FindLargestVersionSinceBaseline(ctx context.Context, tx *sql.Tx, namespace string) (*string, error) {
	largestBaselineSequence, err := driver.FindLargestSequence(ctx, tx, namespace, true /* baseline */)
	if err != nil {
		return nil, err
	}
	const getLargestVersionSinceLastBaselineQuery = `
		SELECT MAX(version) FROM bytebase.migration_history
		WHERE namespace = $1 AND sequence >= $2
	`
	var version sql.NullString
	if err := tx.QueryRowContext(ctx, getLargestVersionSinceLastBaselineQuery,
		namespace, largestBaselineSequence,
	).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(getLargestVersionSinceLastBaselineQuery)
		}
		return nil, util.FormatErrorWithQuery(err, getLargestVersionSinceLastBaselineQuery)
	}
	if version.Valid {
		return &version.String, nil
	}
	return nil, nil
}

// FindLargestSequence will return the largest sequence number.
func (Driver) FindLargestSequence(ctx context.Context, tx *sql.Tx, namespace string, baseline bool) (int, error) {
	findLargestSequenceQuery := `
		SELECT MAX(sequence) FROM bytebase.migration_history
		WHERE namespace = $1`
	if baseline {
		findLargestSequenceQuery = fmt.Sprintf("%s AND (type = '%s' OR type = '%s')", findLargestSequenceQuery, db.Baseline, db.Branch)
	}
	var sequence sql.NullInt32
	if err := tx.QueryRowContext(ctx, findLargestSequenceQuery,
		namespace,
	).Scan(&sequence); err != nil {
		if err == sql.ErrNoRows {
			return -1, common.FormatDBErrorEmptyRowWithQuery(findLargestSequenceQuery)
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
func (Driver) InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (string, error) {
	const insertHistoryQuery = `
	INSERT INTO bytebase.migration_history (
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
		` + "`schema`," + `
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	var insertedID string
	maxIDQuery := "SELECT MAX(id)+1 FROM bytebase.migration_history"
	rows, err := tx.QueryContext(ctx, maxIDQuery)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, maxIDQuery)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(
			&insertedID,
		); err != nil {
			return "", util.FormatErrorWithQuery(err, maxIDQuery)
		}
	}
	if err := rows.Err(); err != nil {
		return "", util.FormatErrorWithQuery(err, maxIDQuery)
	}
	// Clickhouse sql driver doesn't support taking now() as prepared value.
	now := time.Now().Unix()
	_, err = tx.ExecContext(ctx, insertHistoryQuery,
		insertedID,
		m.Creator,
		now,
		m.Creator,
		now,
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
		0,
		m.IssueID,
		m.Payload,
	)
	if err != nil {
		return "", util.FormatErrorWithQuery(err, insertHistoryQuery)
	}

	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID string) error {
	const updateHistoryAsDoneQuery = `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			status = $1,
			execution_duration_ns = $2,
		` + "`schema` = $3" + `
		WHERE id = $4
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsDoneQuery, db.Done, migrationDurationNs, updatedSchema, insertedID)
	return err
}

// UpdateHistoryAsFailed will update the migration record as failed.
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID string) error {
	const updateHistoryAsFailedQuery = `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			status = $1,
			execution_duration_ns = $2
		WHERE id = $3
	`
	_, err := tx.ExecContext(ctx, updateHistoryAsFailedQuery, db.Failed, migrationDurationNs, insertedID)
	return err
}

// ExecuteMigration will execute the migration.
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (string, string, error) {
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
		` + "`schema`," + `
		schema_prev,
		execution_duration_ns,
		issue_id,
		payload
		FROM bytebase.migration_history `
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
	return util.FindMigrationHistoryList(ctx, query, params, driver, db.BytebaseDatabase)
}
