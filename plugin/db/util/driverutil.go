// Package util provides the library for database drivers.
package util

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
)

// FormatErrorWithQuery will format the error with failed query.
func FormatErrorWithQuery(err error, query string) error {
	return common.Wrapf(err, common.DbExecutionError, "failed to execute query %q", query)
}

// ApplyMultiStatements will apply the split statements from scanner.
// This function only used for SQLite, snowflake and clickhouse.
// For MySQL and PostgreSQL, use parser.SplitMultiSQLStream instead.
func ApplyMultiStatements(sc io.Reader, f func(string) error) error {
	// TODO(rebelice): use parser/tokenizer to split SQL statements.
	reader := bufio.NewReader(sc)
	var sb strings.Builder
	delimiter := false
	comment := false
	done := false
	for !done {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				done = true
			} else {
				return err
			}
		}
		line = strings.TrimRight(line, "\r\n")

		execute := false
		switch {
		case strings.HasPrefix(line, "/*"):
			if strings.Contains(line, "*/") {
				if !strings.HasSuffix(line, "*/") {
					return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
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
				return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
			}
			comment = false
			continue
		case sb.Len() == 0 && line == "":
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
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			if !delimiter {
				execute = true
			}
		default:
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			continue
		}
		if execute {
			s := sb.String()
			s = strings.Trim(s, "\n\t ")
			if s != "" {
				if err := f(s); err != nil {
					return errors.Wrapf(err, "execute query %q failed", s)
				}
			}
			sb.Reset()
		}
	}
	// Apply the remaining content.
	s := sb.String()
	s = strings.Trim(s, "\n\t ")
	if s != "" {
		if err := f(s); err != nil {
			return errors.Wrapf(err, "execute query %q failed", s)
		}
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
	if err := rows.Err(); err != nil {
		return false, err
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
func ExecuteMigration(ctx context.Context, executor MigrationExecutor, m *db.MigrationInfo, statement string, databaseName string) (migrationHistoryID int64, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't exist yet.
	if !m.CreateDatabase {
		// For baseline migration, we also record the live schema to detect the schema drift.
		// See https://bytebase.com/blog/what-is-database-schema-drift
		if _, err := executor.Dump(ctx, m.Database, &prevSchemaBuf, true /*schemaOnly*/); err != nil {
			return -1, "", FormatError(err)
		}
	}

	// Phase 1 - Pre-check before executing migration
	// Phase 2 - Record migration history as PENDING
	insertedID, err := BeginMigration(ctx, executor, m, prevSchemaBuf.String(), statement, databaseName)
	if err != nil {
		if common.ErrorCode(err) == common.MigrationAlreadyApplied {
			return insertedID, prevSchemaBuf.String(), nil
		}
		return -1, "", errors.Wrapf(err, "failed to begin migration for issue %s", m.IssueID)
	}

	startedNs := time.Now().UnixNano()

	defer func() {
		if err := EndMigration(ctx, executor, startedNs, insertedID, updatedSchema, databaseName, resErr == nil /*isDone*/); err != nil {
			log.Error("Failed to update migration history record",
				zap.Error(err),
				zap.Int64("migration_id", migrationHistoryID),
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
		if !m.CreateDatabase {
			if _, err := executor.GetDBConnection(ctx, m.Database); err != nil {
				return -1, "", err
			}
		}
		if _, err := executor.Execute(ctx, statement, m.CreateDatabase); err != nil {
			return -1, "", FormatError(err)
		}
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if _, err := executor.Dump(ctx, m.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		return -1, "", FormatError(err)
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// BeginMigration checks before executing migration and inserts a migration history record with pending status.
func BeginMigration(ctx context.Context, executor MigrationExecutor, m *db.MigrationInfo, prevSchema string, statement string, databaseName string) (insertedID int64, err error) {
	// Convert version to stored version.
	storedVersion, err := ToStoredVersion(m.UseSemanticVersion, m.Version, m.SemanticVersionSuffix)
	if err != nil {
		return 0, errors.Wrap(err, "failed to convert to stored version")
	}
	// Phase 1 - Pre-check before executing migration
	// Check if the same migration version has already been applied.
	if list, err := executor.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
		Database: &m.Namespace,
		Version:  &m.Version,
	}); err != nil {
		return -1, errors.Wrap(err, "failed to check duplicate version")
	} else if len(list) > 0 {
		migrationHistory := list[0]
		switch migrationHistory.Status {
		case db.Done:
			if migrationHistory.IssueID != m.IssueID {
				return int64(migrationHistory.ID), common.Errorf(common.MigrationFailed, "database %q has already applied version %s by issue %s", m.Database, m.Version, migrationHistory.IssueID)
			}
			return int64(migrationHistory.ID), common.Errorf(common.MigrationAlreadyApplied, "database %q has already applied version %s", m.Database, m.Version)
		case db.Pending:
			err := errors.Errorf("database %q version %s migration is already in progress", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return int64(migrationHistory.ID), nil
			}
			return -1, common.Wrap(err, common.MigrationPending)
		case db.Failed:
			err := errors.Errorf("database %q version %s migration has failed, please check your database to make sure things are fine and then start a new migration using a new version ", m.Database, m.Version)
			log.Debug(err.Error())
			// For force migration, we will ignore the existing migration history and continue to migration.
			if m.Force {
				return int64(migrationHistory.ID), nil
			}
			return -1, common.Wrap(err, common.MigrationFailed)
		}
	}

	sqldb, err := executor.GetDBConnection(ctx, databaseName)
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
		return -1, common.Errorf(common.MigrationOutOfOrder, "database %q has already applied version %s which >= %s", m.Database, *version, m.Version)
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
func EndMigration(ctx context.Context, executor MigrationExecutor, startedNs int64, migrationHistoryID int64, updatedSchema string, databaseName string, isDone bool) (err error) {
	migrationDurationNs := time.Now().UnixNano() - startedNs

	sqldb, err := executor.GetDBConnection(ctx, databaseName)
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

	return tx.Commit()
}

// Query will execute a readonly / SELECT query.
func Query(ctx context.Context, dbType db.Type, sqldb *sql.DB, statement string, queryContext *db.QueryContext) ([]interface{}, error) {
	readOnly := queryContext.ReadOnly
	limit := queryContext.Limit
	if !readOnly {
		return queryAdmin(ctx, sqldb, statement, limit)
	}
	// Limit SQL query result size.
	if dbType == db.MySQL {
		// MySQL 5.7 doesn't support WITH clause.
		statement = getMySQLStatementWithResultLimit(statement, limit)
	} else {
		statement = getStatementWithResultLimit(statement, limit)
	}

	// TiDB doesn't support READ ONLY transactions. We have to skip the flag for it.
	// https://github.com/pingcap/tidb/issues/34626
	// Clickhouse doesn't support READ ONLY transactions (Error: sql: driver does not support read-only transactions).
	if dbType == db.TiDB || dbType == db.ClickHouse {
		readOnly = false
	}
	tx, err := sqldb.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
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

	fieldList, err := extractSensitiveField(dbType, statement, queryContext.CurrentDatabase, queryContext.SensitiveSchemaInfo)
	if err != nil {
		return nil, err
	}

	if len(fieldList) != 0 && len(fieldList) != len(columnNames) {
		return nil, errors.Errorf("failed to extract sensitive fields: %q", statement)
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
			if len(fieldList) > 0 && fieldList[i].Sensitive {
				rowData = append(rowData, "******")
				continue
			}
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
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return []interface{}{columnNames, columnTypeNames, data}, nil
}

// query will execute a query.
func queryAdmin(ctx context.Context, sqldb *sql.DB, statement string, _ int) ([]interface{}, error) {
	rows, err := sqldb.QueryContext(ctx, statement)
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

	// TODO(d): share the common code.
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
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return []interface{}{columnNames, columnTypeNames, data}, nil
}

func getStatementWithResultLimit(stmt string, limit int) string {
	stmt = strings.TrimRight(stmt, " \n\t;")
	if !strings.HasPrefix(stmt, "EXPLAIN") {
		limitPart := ""
		if limit > 0 {
			limitPart = fmt.Sprintf(" LIMIT %d", limit)
		}
		return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result%s;", stmt, limitPart)
	}
	return stmt
}

func getMySQLStatementWithResultLimit(stmt string, limit int) string {
	stmt = strings.TrimRight(stmt, " \n\t;")
	if !strings.HasPrefix(stmt, "EXPLAIN") {
		limitPart := ""
		if limit > 0 {
			limitPart = fmt.Sprintf(" LIMIT %d", limit)
		}
		return fmt.Sprintf("SELECT * FROM (%s) result%s;", stmt, limitPart)
	}
	return stmt
}

// FindMigrationHistoryList will find the list of migration history.
func FindMigrationHistoryList(ctx context.Context, findMigrationHistoryListQuery string, queryParams []interface{}, driver db.Driver, database string) ([]*db.MigrationHistory, error) {
	// To support `pg` option, the util layer will not know which database where `migration_history` table is,
	// so we need to connect to the database provided by params.
	sqldb, err := driver.GetDBConnection(ctx, database)
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

// FormatError formats schema migration errors.
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "bytebase_idx_unique_migration_history_namespace_version") {
		return errors.Errorf("version has already been applied")
	} else if strings.Contains(err.Error(), "bytebase_idx_unique_migration_history_namespace_sequence") {
		return errors.Errorf("concurrent migration")
	}

	return err
}

// NonSemanticPrefix is the prefix for non-semantic version.
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
		return "", errors.Errorf("invalid version %q, major, minor, patch version should be < 10000", version)
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
		return false, "", "", errors.Errorf("invalid stored version %q, version should contain '-'", storedVersion)
	}
	prefix, suffix := storedVersion[:idx], storedVersion[idx+1:]
	parts := strings.Split(prefix, ".")
	if len(parts) != 3 {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, "", "", errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	if major >= 10000 || minor >= 10000 || patch >= 10000 {
		return false, "", "", errors.Errorf("invalid stored version %q, major, minor, patch version of %q should be < 10000", storedVersion, prefix)
	}
	return true, fmt.Sprintf("%d.%d.%d", major, minor, patch), suffix, nil
}
