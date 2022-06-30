package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	// embed will embeds the migration schema.
	_ "embed"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"go.uber.org/zap"
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
		return nil
	}

	if setup {
		log.Info("Bytebase migration schema not found, creating schema...",
			zap.String("environment", driver.connectionCtx.EnvironmentName),
			zap.String("database", driver.connectionCtx.InstanceName),
		)
		if err := driver.Execute(ctx, migrationSchema); err != nil {
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
		SELECT MAX(version) FROM bytebase.migration_history
		WHERE namespace = $1 AND sequence >= $2
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
		SELECT MAX(sequence) FROM bytebase.migration_history
		WHERE namespace = $1`
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
	var insertedID int64
	maxIDQuery := "SELECT MAX(id)+1 FROM bytebase.migration_history"
	rows, err := tx.QueryContext(ctx, maxIDQuery)
	if err != nil {
		return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(
			&insertedID,
		); err != nil {
			return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
		}
	}
	if err := rows.Err(); err != nil {
		return int64(0), util.FormatErrorWithQuery(err, maxIDQuery)
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
		return int64(0), util.FormatErrorWithQuery(err, insertHistoryQuery)
	}

	return insertedID, nil
}

// UpdateHistoryAsDone will update the migration record as done.
func (Driver) UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error {
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
func (Driver) UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error {
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
func (driver *Driver) ExecuteMigration(ctx context.Context, m *db.MigrationInfo, statement string) (int64, string, error) {
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
		`ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	// TODO(zp):  modified param database of `util.FindMigrationHistoryList` when we support *clickhouse* database level.
	history, err := util.FindMigrationHistoryList(ctx, query, params, driver, db.BytebaseDatabase, find, baseQuery)
	// TODO(d): remove this block once all existing customers all migrated to semantic versioning.
	if err != nil {
		if !strings.Contains(err.Error(), "invalid stored version") {
			return nil, err
		}
		if err := driver.updateMigrationHistoryStorageVersion(ctx); err != nil {
			return nil, err
		}
		return util.FindMigrationHistoryList(ctx, query, params, driver, db.BytebaseDatabase, find, baseQuery)
	}
	return history, err
}
func (driver *Driver) updateMigrationHistoryStorageVersion(ctx context.Context) error {
	sqldb, err := driver.GetDbConnection(ctx, "bytebase")
	if err != nil {
		return err
	}
	query := `SELECT id, version FROM bytebase.migration_history`
	rows, err := sqldb.Query(query)
	if err != nil {
		return err
	}
	type ver struct {
		id      int
		version string
	}
	var vers []ver
	for rows.Next() {
		var v ver
		if err := rows.Scan(&v.id, &v.version); err != nil {
			return err
		}
		vers = append(vers, v)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	updateQuery := `
		ALTER TABLE
			bytebase.migration_history
		UPDATE
			version = $1
		WHERE id = $2 AND version = $3
	`
	for _, v := range vers {
		if strings.HasPrefix(v.version, util.NonSemanticPrefix) {
			continue
		}
		newVersion := fmt.Sprintf("%s%s", util.NonSemanticPrefix, v.version)
		if _, err := sqldb.Exec(updateQuery, newVersion, v.id, v.version); err != nil {
			return err
		}
	}
	return nil
}
