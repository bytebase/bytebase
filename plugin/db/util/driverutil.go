package util

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

// FormatErrorWithQuery will format the error with failed query.
func FormatErrorWithQuery(err error, query string) error {
	return common.Errorf(common.DbExecutionError, fmt.Errorf("failed to execute error: %w\n\nquery:\n%q", err, query))
}

// ApplyMultiStatements will apply the splitted statements from scanner.
func ApplyMultiStatements(sc *bufio.Scanner, f func(string) error) error {
	s := ""
	delimiter := false
	comment := false
	for sc.Scan() {
		line := sc.Text()

		execute := false
		switch {
		case strings.HasPrefix(line, "/*"):
			if strings.Contains(line, "*/") {
				if !strings.HasSuffix(line, "*/") {
					return fmt.Errorf("`*/` must be the end of the line; new statement should start as a new line")
				}
			} else {
				comment = true
			}
			continue
		case comment && !strings.Contains(line, "*/"):
			// Skip the line when in comment mode.
			continue
		case comment && strings.Contains(line, "*/"):
			if !strings.HasSuffix(line, "*/") {
				return fmt.Errorf("`*/` must be the end of the line; new statement should start as a new line")
			}
			comment = false
			continue
		case s == "" && line == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case line == "DELIMITER ;;":
			delimiter = true
			continue
		case line == "DELIMITER ;" && delimiter:
			delimiter = false
			execute = true
		case strings.HasSuffix(line, ";"):
			s = s + line + "\n"
			if !delimiter {
				execute = true
			}
		default:
			s = s + line + "\n"
			continue
		}
		if execute {
			s = strings.Trim(s, "\n\t ")
			if s != "" {
				if err := f(s); err != nil {
					return fmt.Errorf("execute query %q failed: %v", s, err)
				}
			}
			s = ""
		}
	}
	// Apply the remaining content.
	s = strings.Trim(s, "\n\t ")
	if s != "" {
		if err := f(s); err != nil {
			return fmt.Errorf("execute query %q failed: %v", s, err)
		}
	}

	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

// NeedsSetupMigrationSchema will return whether it's needed to setup migration schema.
func NeedsSetupMigrationSchema(ctx context.Context, sqldb *sql.DB, query string) (bool, error) {
	rows, err := sqldb.QueryContext(ctx, query)
	if err != nil {
		return false, FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	if rows.Next() {
		return false, nil
	}

	return true, nil
}

// MigrationExecutor is an adapter for ExecuteMigration().
type MigrationExecutor interface {
	db.Driver
	// FindLargestVersionSinceBaseline will find the largest version since last baseline or branch.
	FindLargestVersionSinceBaseline(ctx context.Context, tx *sql.Tx, namespace string) (*string, error)
	// FindLargestSequence will return the largest sequence number.
	// Returns 0 if we haven't applied any migration for this namespace.
	FindLargestSequence(ctx context.Context, tx *sql.Tx, namespace string, baseline bool) (int, error)
	// InsertPendingHistory will insert the migration record with pending status and return the inserted ID.
	InsertPendingHistory(ctx context.Context, tx *sql.Tx, sequence int, prevSchema string, m *db.MigrationInfo, storedVersion, statement string) (insertedID int64, err error)
	// UpdateHistoryAsDone will update the migration record as done.
	UpdateHistoryAsDone(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, updatedSchema string, insertedID int64) error
	// UpdateHistoryAsFailed will update the migration record as failed.
	UpdateHistoryAsFailed(ctx context.Context, tx *sql.Tx, migrationDurationNs int64, insertedID int64) error
}

// ExecuteMigration will execute the database migration.
// Returns the created migration history id and the updated schema on success.
func ExecuteMigration(ctx context.Context, l *zap.Logger, executor MigrationExecutor, m *db.MigrationInfo, statement string, databaseName string) (migrationHistoryID int64, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't exist yet.
	if !m.CreateDatabase {
		// For baseline migration, we also record the live schema to detect the schema drift.
		// See https://bytebase.com/blog/what-is-database-schema-drift
		if err := executor.Dump(ctx, m.Database, &prevSchemaBuf, true /*schemaOnly*/); err != nil {
			return -1, "", FormatError(err)
		}
	}

	// Phase 1 - Precheck before executing migration
	// Phase 2 - Record migration history as PENDING
	insertedID, err := BeginMigration(ctx, executor, m, prevSchemaBuf.String(), statement, databaseName)
	if err != nil {
		return -1, "", err
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := EndMigration(ctx, l, executor, startedNs, insertedID, updatedSchema, databaseName, resErr == nil /*isDone*/); err != nil {
			l.Error("Failed to update migration history record",
				zap.Error(err),
				zap.Int64("migration_id", migrationHistoryID),
			)
		}
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute, except for CreateDatabase.
	// https://github.com/bytebase/bytebase/issues/394
	if statement != "" && (m.Type != db.Baseline || m.CreateDatabase) {
		// Switch to the target database only if we're NOT creating this target database.
		if !m.CreateDatabase {
			if _, err := executor.GetDbConnection(ctx, m.Database); err != nil {
				return -1, "", err
			}
		}
		if err := executor.Execute(ctx, statement); err != nil {
			return -1, "", FormatError(err)
		}
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if err := executor.Dump(ctx, m.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		return -1, "", FormatError(err)
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func BeginMigration(ctx context.Context, executor MigrationExecutor, m *db.MigrationInfo, prevSchema string, statement string, databaseName string) (insertedID int64, err error) {
	// Convert version to stored version.
	storedVersion, err := ToStoredVersion(m.UseSemanticVersion, m.Version, m.SemanticVersionSuffix)
	if err != nil {
		return 0, fmt.Errorf("failed to convert to stored version, error %w", err)
	}
	// Phase 1 - Precheck before executing migration
	// Check if the same migration version has already been applied
	if list, err := executor.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
		Database: &m.Namespace,
		Version:  &m.Version,
	}); err != nil {
		return -1, fmt.Errorf("check duplicate version error: %q", err)
	} else if len(list) > 0 {
		switch list[0].Status {
		case db.Done:
			return -1, common.Errorf(common.MigrationAlreadyApplied,
				fmt.Errorf("database %q has already applied version %s", m.Database, m.Version))
		case db.Pending:
			return -1, common.Errorf(common.MigrationPending,
				fmt.Errorf("database %q version %s migration is already in progress", m.Database, m.Version))
		case db.Failed:
			return -1, common.Errorf(common.MigrationFailed,
				fmt.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version ", m.Database, m.Version))
		}
	}

	sqldb, err := executor.GetDbConnection(ctx, databaseName)
	if err != nil {
		return -1, err
	}
	// From a concurrency perspective, there's no difference between using transaction or not. However, we use transaction here to save some code of starting a transaction inside each db engine executor.
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	largestSequence, err := executor.FindLargestSequence(ctx, tx, m.Namespace, false /* baseline */)
	if err != nil {
		return -1, err
	}

	// Check if there is any higher version already been applied since the last baseline or branch.
	if version, err := executor.FindLargestVersionSinceBaseline(ctx, tx, m.Namespace); err != nil {
		return -1, err
	} else if version != nil && len(*version) > 0 && *version >= m.Version {
		// len(*version) > 0 is used because Clickhouse will always return non-nil version with empty string.
		return -1, common.Errorf(common.MigrationOutOfOrder, fmt.Errorf("database %q has already applied version %s which >= %s", m.Database, *version, m.Version))
	}

	// Phase 2 - Record migration history as PENDING.
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	if insertedID, err = executor.InsertPendingHistory(ctx, tx, largestSequence+1, prevSchema, m, storedVersion, statement); err != nil {
		return -1, err
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return insertedID, nil
}

// EndMigration updates the migration history record to DONE or FAILED depending on migration is done or not.
func EndMigration(ctx context.Context, l *zap.Logger, executor MigrationExecutor, startedNs int64, migrationHistoryID int64, updatedSchema string, databaseName string, isDone bool) (err error) {
	migrationDurationNs := time.Now().UnixNano() - startedNs

	sqldb, err := executor.GetDbConnection(ctx, databaseName)
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
		err = executor.UpdateHistoryAsDone(ctx, tx, migrationDurationNs, updatedSchema, migrationHistoryID)
	} else {
		// Otherwise, update the migration history as 'FAILED', execution_duration.
		err = executor.UpdateHistoryAsFailed(ctx, tx, migrationDurationNs, migrationHistoryID)
	}

	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Query will execute a readonly / SELECT query.
func Query(ctx context.Context, l *zap.Logger, sqldb *sql.DB, statement string, limit int) ([]interface{}, error) {
	// Not all sql engines support ReadOnly flag, so we will use tx rollback semantics to enforce readonly.
	tx, err := sqldb.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, FormatError(err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, FormatError(err)
	}

	colCount := len(columnTypes)

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	rowCount := 0
	data := []interface{}{}
	for rows.Next() {
		scanArgs := make([]interface{}, colCount)
		for i, v := range columnTypeNames {
			// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
			switch v {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT", "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
			case "FLOAT":
				scanArgs[i] = new(sql.NullFloat64)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, FormatError(err)
		}

		rowData := []interface{}{}
		for i := range columnTypes {
			if v, ok := (scanArgs[i]).(*sql.NullBool); ok && v.Valid {
				rowData = append(rowData, v.Bool)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullString); ok && v.Valid {
				rowData = append(rowData, v.String)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt64); ok && v.Valid {
				rowData = append(rowData, v.Int64)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt32); ok && v.Valid {
				rowData = append(rowData, v.Int32)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullFloat64); ok && v.Valid {
				rowData = append(rowData, v.Float64)
				continue
			}
			// If none of them match, set nil to its value.
			rowData = append(rowData, nil)
		}

		data = append(data, rowData)
		rowCount++
		if rowCount == limit {
			break
		}
	}

	return []interface{}{columnNames, columnTypeNames, data}, nil
}

// FindMigrationHistoryList will find the list of migration history.
func FindMigrationHistoryList(ctx context.Context, findMigrationHistoryListQuery string, queryParams []interface{}, driver db.Driver, database string, find *db.MigrationHistoryFind, baseQuery string) ([]*db.MigrationHistory, error) {
	// To support `pg` option, util layer will not know which database `migration_history` is located in,
	// so wo need connect database provided by params.
	sqldb, err := driver.GetDbConnection(ctx, database)
	if err != nil {
		return nil, err
	}
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, findMigrationHistoryListQuery, queryParams...)
	if err != nil {
		return nil, FormatErrorWithQuery(err, findMigrationHistoryListQuery)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into migrationHistoryList.
	var migrationHistoryList []*db.MigrationHistory
	for rows.Next() {
		var history db.MigrationHistory
		var storedVersion string
		if err := rows.Scan(
			&history.ID,
			&history.Creator,
			&history.CreatedTs,
			&history.Updater,
			&history.UpdatedTs,
			&history.ReleaseVersion,
			&history.Namespace,
			&history.Sequence,
			&history.Source,
			&history.Type,
			&history.Status,
			&storedVersion,
			&history.Description,
			&history.Statement,
			&history.Schema,
			&history.SchemaPrev,
			&history.ExecutionDurationNs,
			&history.IssueID,
			&history.Payload,
		); err != nil {
			return nil, err
		}

		useSemanticVersion, version, semanticVersionSuffix, err := fromStoredVersion(storedVersion)
		if err != nil {
			return nil, err
		}
		history.UseSemanticVersion, history.Version, history.SemanticVersionSuffix = useSemanticVersion, version, semanticVersionSuffix
		migrationHistoryList = append(migrationHistoryList, &history)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return migrationHistoryList, nil
}

// FormatError formats schema migration errors
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "bytebase_idx_unique_migration_history_namespace_version") {
		return fmt.Errorf("version has already been applied")
	} else if strings.Contains(err.Error(), "bytebase_idx_unique_migration_history_namespace_sequence") {
		return fmt.Errorf("concurrent migration")
	}

	return err
}

// NonSemanticPrefix is the prefix for non-semantic version
const NonSemanticPrefix = "0000.0000.0000-"

// ToStoredVersion converts semantic or non-semantic version to stored version format.
// Non-semantic version will have additional "0000.0000.0000-" prefix.
// Semantic version will add zero padding to MAJOR, MINOR, PATCH version with a timestamp suffix.
func ToStoredVersion(useSemanticVersion bool, version, semanticVersionSuffix string) (string, error) {
	if !useSemanticVersion {
		return fmt.Sprintf("%s%s", NonSemanticPrefix, version), nil
	}
	v, err := semver.Make(version)
	if err != nil {
		return "", err
	}
	major, minor, patch := fmt.Sprintf("%d", v.Major), fmt.Sprintf("%d", v.Minor), fmt.Sprintf("%d", v.Patch)
	if len(major) > 4 || len(minor) > 4 || len(patch) > 4 {
		return "", fmt.Errorf("invalid version %q, major, minor, patch version should be < 10000", version)
	}
	return fmt.Sprintf("%04s.%04s.%04s-%s", major, minor, patch, semanticVersionSuffix), nil
}

// fromStoredVersion converts stored version to semantic or non-semantic version.
func fromStoredVersion(storedVersion string) (bool, string, string, error) {
	if strings.HasPrefix(storedVersion, NonSemanticPrefix) {
		return false, strings.TrimPrefix(storedVersion, NonSemanticPrefix), "", nil
	}
	idx := strings.Index(storedVersion, "-")
	if idx < 0 {
		return false, "", "", fmt.Errorf("invalid stored version %q, version should contain '-'", storedVersion)
	}
	prefix, suffix := storedVersion[:idx], storedVersion[idx+1:]
	parts := strings.Split(prefix, ".")
	if len(parts) != 3 {
		return false, "", "", fmt.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, "", "", fmt.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, "", "", fmt.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, "", "", fmt.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	if major >= 10000 || minor >= 10000 || patch >= 10000 {
		return false, "", "", fmt.Errorf("invalid stored version %q, major, minor, patch version of %q should be < 10000", storedVersion, prefix)
	}
	return true, fmt.Sprintf("%d.%d.%d", major, minor, patch), suffix, nil
}
