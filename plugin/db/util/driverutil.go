package util

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

const (
	bytebaseDatabase = "bytebase"
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

// RecordPendingMigrationHistory is a helper function to implement MigrationExecutableDriver.RecordPendingMigrationHistory() for MySQL.
func RecordPendingMigrationHistory(ctx context.Context, l *zap.Logger, tx *sql.Tx, m *db.MigrationInfo, statement string, sequence int, prevSchema string) (insertedID int64, err error) {
	const insertHistoryQuery = `
	INSERT INTO bytebase.migration_history (
		created_by,
		created_ts,
		updated_by,
		updated_ts,
		release_version,
		namespace,
		sequence,
		engine,
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
	VALUES (?, unix_timestamp(), ?, unix_timestamp(), ?, ?, ?, ?,  ?, 'PENDING', ?, ?, ?, ?, ?, 0, ?, ?)
`
	res, err := tx.ExecContext(ctx, insertHistoryQuery,
		m.Creator,
		m.Creator,
		m.ReleaseVersion,
		m.Namespace,
		sequence,
		m.Engine,
		m.Type,
		m.Version,
		m.Description,
		statement,
		prevSchema,
		prevSchema,
		m.IssueID,
		m.Payload,
	)

	if err != nil {
		return -1, FormatErrorWithQuery(err, insertHistoryQuery)
	}

	insertedID, err = res.LastInsertId()
	if err != nil {
		return -1, FormatErrorWithQuery(err, insertHistoryQuery)
	}
	return insertedID, nil
}

// MigrationExecutionArgs includes the arguments for ExecuteMigration().
type MigrationExecutionArgs struct {
	InsertHistoryQuery         string
	UpdateHistoryAsDoneQuery   string
	UpdateHistoryAsFailedQuery string
	TablePrefix                string
}

// MigrationExecutableDriver is an adapter for db.Driver and db.util.ExecuteMigration.
type MigrationExecutableDriver interface {
	db.Driver
	RecordPendingMigrationHistory(ctx context.Context, l *zap.Logger, tx *sql.Tx, m *db.MigrationInfo, statement string, sequence int, prevSchema string) (insertedID int64, err error)
}

// ExecuteMigration will execute the database migration.
// Returns the created migraiton history id and the updated schema on success.
func ExecuteMigration(ctx context.Context, l *zap.Logger, dbType db.Type, driver MigrationExecutableDriver, m *db.MigrationInfo, statement string, args MigrationExecutionArgs) (migrationHistoryID int64, updatedSchema string, resErr error) {
	var prevSchemaBuf bytes.Buffer
	// Don't record schema if the database hasn't exist yet.
	if !m.CreateDatabase {
		if err := driver.Dump(ctx, m.Database, &prevSchemaBuf, true /*schemaOnly*/); err != nil {
			return -1, "", formatError(err)
		}
	}

	sqldb, err := driver.GetDbConnection(ctx, bytebaseDatabase)
	if err != nil {
		return -1, "", err
	}
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return -1, "", err
	}
	defer tx.Rollback()

	// Phase 1 - Precheck before executing migration
	// Check if the same migration version has alraedy been applied
	duplicate, err := checkDuplicateVersion(ctx, driver, tx, m.Namespace, m.Engine, m.Version, args.TablePrefix)
	if err != nil {
		return -1, "", err
	}
	if duplicate {
		return -1, "", common.Errorf(common.MigrationAlreadyApplied, fmt.Errorf("database %q has already applied version %s", m.Database, m.Version))
	}

	// Check if there is any higher version already been applied
	version, err := checkOutofOrderVersion(ctx, driver, tx, m.Namespace, m.Engine, m.Version, args.TablePrefix)
	if err != nil {
		return -1, "", err
	}
	// Clickhouse will always return non-nil version with empty string.
	if version != nil && len(*version) > 0 {
		return -1, "", common.Errorf(common.MigrationOutOfOrder, fmt.Errorf("database %q has already applied version %s which is higher than %s", m.Database, *version, m.Version))
	}

	// If the migration engine is VCS and type is not baseline and is not branch, then we can only proceed if there is existing baseline
	// This check is also wrapped in transaction to avoid edge case where two baselinings are running concurrently.
	if m.Engine == db.VCS && m.Type != db.Baseline && m.Type != db.Branch {
		hasBaseline, err := findBaseline(ctx, driver, tx, m.Namespace, args.TablePrefix)
		if err != nil {
			return -1, "", err
		}

		if !hasBaseline {
			return -1, "", common.Errorf(common.MigrationBaselineMissing, fmt.Errorf("%s has not created migration baseline yet", m.Database))
		}
	}

	// VCS based SQL migration requires existing baselining
	requireBaseline := m.Engine == db.VCS && m.Type == db.Migrate
	sequence, err := findNextSequence(ctx, driver, tx, m.Namespace, requireBaseline, args.TablePrefix)
	if err != nil {
		return -1, "", err
	}

	// Phase 2 - Record migration history as PENDING
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	insertedID, err := driver.RecordPendingMigrationHistory(ctx, l, tx, m, statement, sequence, prevSchemaBuf.String())
	if err != nil {
		return -1, "", err
	}

	if err := tx.Commit(); err != nil {
		return -1, "", err
	}

	startedNs := time.Now().UnixNano()

	// If we have already started migration, there will be a PENDING migration record. Upon returning,
	// we will update that record to DONE or FAILED depending on whether error occurs.
	defer func() (myErr error) {
		defer func() {
			if myErr != nil {
				l.Error("Failed to update migration history record",
					zap.Error(myErr),
					zap.Int64("migration_id", insertedID),
				)
			}
		}()

		migrationDurationNs := time.Now().UnixNano() - startedNs
		afterSqldb, tmpErr := driver.GetDbConnection(ctx, bytebaseDatabase)
		if tmpErr != nil {
			return
		}
		afterTx, tmpErr := afterSqldb.BeginTx(ctx, nil)
		if tmpErr != nil {
			return
		}
		defer afterTx.Rollback()

		if resErr == nil {
			// Upon success, update the migration history as 'DONE', execution_duration_ns, updated schema.
			_, tmpErr = afterTx.ExecContext(ctx, args.UpdateHistoryAsDoneQuery,
				migrationDurationNs,
				updatedSchema,
				insertedID,
			)
		} else {
			// Otherwise, update the migration history as 'FAILED', exeuction_duration
			_, tmpErr = afterTx.ExecContext(ctx, args.UpdateHistoryAsFailedQuery,
				migrationDurationNs,
				insertedID,
			)
		}

		if tmpErr != nil {
			return tmpErr
		}

		return afterTx.Commit()
	}()

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could has non-empty sql but will not execute, except for CreateDatabase.
	// https://github.com/bytebase/bytebase/issues/394
	if statement != "" && (m.Type != db.Baseline || m.CreateDatabase) {
		// Switch to the target database only if we're NOT creating this target database.
		if !m.CreateDatabase {
			_, err := driver.GetDbConnection(ctx, m.Database)
			if err != nil {
				return -1, "", err
			}
		}
		// MySQL executes DDL in its own transaction, so there is no need to supply a transaction from previous migration history updates.
		// Also, we don't use transaction for creating databases in Postgres.
		// https://github.com/bytebase/bytebase/issues/202
		if err = driver.Execute(ctx, statement, !m.CreateDatabase); err != nil {
			return -1, "", formatError(err)
		}
	}

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if err = driver.Dump(ctx, m.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		return -1, "", formatError(err)
	}

	return insertedID, afterSchemaBuf.String(), nil
}

// Query will execute a readonly / SELECT query.
func Query(ctx context.Context, l *zap.Logger, db *sql.DB, statement string, limit int) ([]interface{}, error) {
	// Not all sql engines support ReadOnly flag, so we will use tx rollback semantics to enforce readonly.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, formatError(err)
	}

	colCount := len(columnTypes)
	rowCount := 0
	resultSet := []interface{}{}
	for rows.Next() {
		scanArgs := make([]interface{}, colCount)
		for i, v := range columnTypes {
			switch v.DatabaseTypeName() {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT4":
				scanArgs[i] = new(sql.NullInt64)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, formatError(err)
		}

		rowData := map[string]interface{}{}
		for i, v := range columnTypes {
			if z, ok := (scanArgs[i]).(*sql.NullBool); ok {
				rowData[v.Name()] = z.Bool
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				rowData[v.Name()] = z.String
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullInt64); ok {
				rowData[v.Name()] = z.Int64
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				rowData[v.Name()] = z.Float64
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullInt32); ok {
				rowData[v.Name()] = z.Int32
				continue
			}
			rowData[v.Name()] = scanArgs[i]
		}

		resultSet = append(resultSet, rowData)
		rowCount++
		if rowCount == limit {
			break
		}
	}

	return resultSet, nil
}

// QueryString is a helper function to implement Driver.QueryString() for MySQL.
func QueryString(p *db.QueryParams) string {
	params := p.Names
	if len(params) == 0 {
		return ""
	}
	for i, param := range params {
		if !strings.Contains(param, "?") {
			params[i] = param + " = ?"
		}
	}
	return fmt.Sprintf("WHERE %s ", strings.Join(params, " AND "))
}

func findBaseline(ctx context.Context, driver db.Driver, tx *sql.Tx, namespace, tablePrefix string) (bool, error) {
	var queryParams db.QueryParams
	queryParams.AddParam("namespace", namespace)
	queryParams.AddParam("type", "BASELINE")
	query := `
		SELECT 1 FROM ` +
		tablePrefix + `migration_history ` +
		driver.QueryString(&queryParams)
	row, err := tx.QueryContext(ctx, query,
		queryParams.Params...,
	)

	if err != nil {
		return false, FormatErrorWithQuery(err, query)
	}
	defer row.Close()

	if !row.Next() {
		return false, nil
	}

	return true, nil
}

func checkDuplicateVersion(ctx context.Context, driver db.Driver, tx *sql.Tx, namespace string, engine db.MigrationEngine, version, tablePrefix string) (bool, error) {
	var queryParams db.QueryParams
	queryParams.AddParam("namespace", namespace)
	queryParams.AddParam("engine", engine.String())
	queryParams.AddParam("version", version)
	query := `
		SELECT 1 FROM ` +
		tablePrefix + `migration_history ` +
		driver.QueryString(&queryParams)
	row, err := tx.QueryContext(ctx, query,
		queryParams.Params...,
	)

	if err != nil {
		return false, FormatErrorWithQuery(err, query)
	}
	defer row.Close()

	if row.Next() {
		return true, nil
	}
	return false, nil
}

func checkOutofOrderVersion(ctx context.Context, driver db.Driver, tx *sql.Tx, namespace string, engine db.MigrationEngine, version, tablePrefix string) (*string, error) {
	var queryParams db.QueryParams
	queryParams.AddParam("namespace", namespace)
	queryParams.AddParam("engine", engine.String())
	queryParams.AddParam("version > ?", version)
	query := `
		SELECT MIN(version) FROM ` +
		tablePrefix + `migration_history ` +
		driver.QueryString(&queryParams)
	row, err := tx.QueryContext(ctx, query,
		queryParams.Params...,
	)

	if err != nil {
		return nil, FormatErrorWithQuery(err, query)
	}
	defer row.Close()

	var minVersion sql.NullString
	row.Next()
	if err := row.Scan(&minVersion); err != nil {
		return nil, err
	}

	if minVersion.Valid {
		return &minVersion.String, nil
	}

	return nil, nil
}

func findNextSequence(ctx context.Context, driver db.Driver, tx *sql.Tx, namespace string, requireBaseline bool, tablePrefix string) (int, error) {
	var queryParams db.QueryParams
	queryParams.AddParam("namespace", namespace)

	query := `
		SELECT MAX(sequence) + 1 FROM ` +
		tablePrefix + `migration_history ` +
		driver.QueryString(&queryParams)
	row, err := tx.QueryContext(ctx, query,
		queryParams.Params...,
	)

	if err != nil {
		return -1, FormatErrorWithQuery(err, query)
	}
	defer row.Close()

	var sequence sql.NullInt32
	row.Next()
	if err := row.Scan(&sequence); err != nil {
		return -1, err
	}

	if !sequence.Valid {
		// Returns 1 if we haven't applied any migration for this namespace and doesn't require baselining
		if !requireBaseline {
			return 1, nil
		}

		// This should not happen normally since we already check the baselining exist beforehand. Just in case.
		return -1, common.Errorf(common.MigrationBaselineMissing, fmt.Errorf("unable to generate next migration_sequence, no migration hisotry found for %q, do you forget to baselining?", namespace))
	}

	return int(sequence.Int32), nil
}

// FindMigrationHistoryList will find the list of migration history.
func FindMigrationHistoryList(ctx context.Context, driver db.Driver, find *db.MigrationHistoryFind, baseQuery string) ([]*db.MigrationHistory, error) {
	sqldb, err := driver.GetDbConnection(ctx, bytebaseDatabase)
	if err != nil {
		return nil, err
	}
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var queryParams db.QueryParams
	if v := find.ID; v != nil {
		queryParams.AddParam("id", *v)
	}
	if v := find.Database; v != nil {
		queryParams.AddParam("namespace", *v)
	}
	if v := find.Version; v != nil {
		queryParams.AddParam("version", *v)
	}

	var query = baseQuery +
		driver.QueryString(&queryParams) +
		`ORDER BY created_ts DESC`
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}

	rows, err := tx.QueryContext(ctx, query, queryParams.Params...)
	if err != nil {
		return nil, FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*db.MigrationHistory, 0)
	for rows.Next() {
		var history db.MigrationHistory
		if err := rows.Scan(
			&history.ID,
			&history.Creator,
			&history.CreatedTs,
			&history.Updater,
			&history.UpdatedTs,
			&history.ReleaseVersion,
			&history.Namespace,
			&history.Sequence,
			&history.Engine,
			&history.Type,
			&history.Status,
			&history.Version,
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

		list = append(list, &history)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return list, nil
}

func formatError(err error) error {
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
