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

// MigrationExecutionArgs includes the arguments for ExecuteMigration().
type MigrationExecutionArgs struct {
	InsertHistoryQuery         string
	UpdateHistoryAsDoneQuery   string
	UpdateHistoryAsFailedQuery string
	TablePrefix                string
}

// ExecuteMigration will execute the database migration.
// Returns the created migraiton history id and the updated schema on success.
func ExecuteMigration(ctx context.Context, l *zap.Logger, dbType db.Type, driver db.Driver, m *db.MigrationInfo, statement string, args MigrationExecutionArgs) (migrationHistoryID int64, updatedSchema string, resErr error) {
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
	duplicate, err := checkDuplicateVersion(ctx, dbType, tx, m.Namespace, m.Engine, m.Version, args.TablePrefix)
	if err != nil {
		return -1, "", err
	}
	if duplicate {
		return -1, "", common.Errorf(common.MigrationAlreadyApplied, fmt.Errorf("database %q has already applied version %s", m.Database, m.Version))
	}

	// Check if there is any higher version already been applied
	version, err := checkOutofOrderVersion(ctx, dbType, tx, m.Namespace, m.Engine, m.Version, args.TablePrefix)
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
		hasBaseline, err := findBaseline(ctx, dbType, tx, m.Namespace, args.TablePrefix)
		if err != nil {
			return -1, "", err
		}

		if !hasBaseline {
			return -1, "", common.Errorf(common.MigrationBaselineMissing, fmt.Errorf("%s has not created migration baseline yet", m.Database))
		}
	}

	// VCS based SQL migration requires existing baselining
	requireBaseline := m.Engine == db.VCS && m.Type == db.Migrate
	sequence, err := findNextSequence(ctx, dbType, tx, m.Namespace, requireBaseline, args.TablePrefix)
	if err != nil {
		return -1, "", err
	}

	// Phase 2 - Record migration history as PENDING
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	insertedID := int64(-1)
	if dbType == db.Postgres {
		tx.QueryRowContext(ctx, args.InsertHistoryQuery,
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
			prevSchemaBuf.String(),
			prevSchemaBuf.String(),
			m.IssueID,
			m.Payload,
		).Scan(&insertedID)
	} else if dbType == db.ClickHouse {
		maxIDQuery := "SELECT MAX(id)+1 FROM bytebase.migration_history"
		rows, err := tx.QueryContext(ctx, maxIDQuery)
		if err != nil {
			return -1, "", FormatErrorWithQuery(err, maxIDQuery)
		}
		defer rows.Close()
		for rows.Next() {
			if err := rows.Scan(
				&insertedID,
			); err != nil {
				return -1, "", FormatErrorWithQuery(err, maxIDQuery)
			}
		}
		if err := rows.Err(); err != nil {
			return -1, "", FormatErrorWithQuery(err, maxIDQuery)
		}
		// Clickhouse sql driver doesn't support taking now() as prepared value.
		now := time.Now().Unix()
		_, err = tx.ExecContext(ctx, args.InsertHistoryQuery,
			insertedID,
			m.Creator,
			now,
			m.Creator,
			now,
			m.ReleaseVersion,
			m.Namespace,
			sequence,
			m.Engine,
			m.Type,
			"PENDING",
			m.Version,
			m.Description,
			statement,
			prevSchemaBuf.String(),
			prevSchemaBuf.String(),
			0,
			m.IssueID,
			m.Payload,
		)
		if err != nil {
			return -1, "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
		}
	} else if dbType == db.Snowflake {
		maxIDQuery := "SELECT MAX(id)+1 FROM bytebase.public.migration_history"
		rows, err := tx.QueryContext(ctx, maxIDQuery)
		if err != nil {
			return -1, "", FormatErrorWithQuery(err, maxIDQuery)
		}
		defer rows.Close()
		var id sql.NullInt64
		for rows.Next() {
			if err := rows.Scan(
				&id,
			); err != nil {
				return -1, "", FormatErrorWithQuery(err, maxIDQuery)
			}
		}
		if err := rows.Err(); err != nil {
			return -1, "", FormatErrorWithQuery(err, maxIDQuery)
		}
		if id.Valid {
			insertedID = id.Int64
		} else {
			insertedID = 1
		}

		_, err = tx.ExecContext(ctx, args.InsertHistoryQuery,
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
			prevSchemaBuf.String(),
			prevSchemaBuf.String(),
			m.IssueID,
			m.Payload,
		)
		if err != nil {
			return -1, "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
		}
	} else {
		res, err := tx.ExecContext(ctx, args.InsertHistoryQuery,
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
			prevSchemaBuf.String(),
			prevSchemaBuf.String(),
			m.IssueID,
			m.Payload,
		)

		if err != nil {
			return -1, "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
		}

		insertedID, err = res.LastInsertId()
		if err != nil {
			return -1, "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
		}
	}

	if err := tx.Commit(); err != nil {
		return -1, "", err
	}

	startedTs := time.Now().Unix()

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

		migrationDuration := time.Now().Unix() - startedTs
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
			// Upon success, update the migration history as 'DONE', execution_duration, updated schema.
			_, tmpErr = afterTx.ExecContext(ctx, args.UpdateHistoryAsDoneQuery,
				migrationDuration,
				updatedSchema,
				insertedID,
			)
		} else {
			// Otherwise, update the migration history as 'FAILED', exeuction_duration
			_, tmpErr = afterTx.ExecContext(ctx, args.UpdateHistoryAsFailedQuery,
				migrationDuration,
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
	// Baseline migration type could also has empty sql when the database is newly created.
	if statement != "" {
		// Switch to the target database only if we're NOT creating this target database.
		if !m.CreateDatabase {
			_, err := driver.GetDbConnection(ctx, m.Database)
			if err != nil {
				return -1, "", err
			}
		}
		// MySQL executes DDL in its own transaction, so there is no need to supply a transaction.
		if err = driver.Execute(ctx, statement); err != nil {
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

func findBaseline(ctx context.Context, dbType db.Type, tx *sql.Tx, namespace, tablePrefix string) (bool, error) {
	queryParams := &db.QueryParams{DatabaseType: dbType}
	queryParams.AddParam("namespace", namespace)
	queryParams.AddParam("type", "BASELINE")
	query := `
		SELECT 1 FROM ` +
		tablePrefix + `migration_history ` +
		queryParams.QueryString()
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

func checkDuplicateVersion(ctx context.Context, dbType db.Type, tx *sql.Tx, namespace string, engine db.MigrationEngine, version, tablePrefix string) (bool, error) {
	queryParams := &db.QueryParams{DatabaseType: dbType}
	queryParams.AddParam("namespace", namespace)
	queryParams.AddParam("engine", engine.String())
	queryParams.AddParam("version", version)
	query := `
		SELECT 1 FROM ` +
		tablePrefix + `migration_history ` +
		queryParams.QueryString()
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

func checkOutofOrderVersion(ctx context.Context, dbType db.Type, tx *sql.Tx, namespace string, engine db.MigrationEngine, version, tablePrefix string) (*string, error) {
	queryParams := &db.QueryParams{DatabaseType: dbType}
	queryParams.AddParam("namespace", namespace)
	queryParams.AddParam("engine", engine.String())
	queryParams.AddParam("version > ?", version)
	query := `
		SELECT MIN(version) FROM ` +
		tablePrefix + `migration_history ` +
		queryParams.QueryString()
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

func findNextSequence(ctx context.Context, dbType db.Type, tx *sql.Tx, namespace string, requireBaseline bool, tablePrefix string) (int, error) {
	queryParams := &db.QueryParams{DatabaseType: dbType}
	queryParams.AddParam("namespace", namespace)

	query := `
		SELECT MAX(sequence) + 1 FROM ` +
		tablePrefix + `migration_history ` +
		queryParams.QueryString()
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
func FindMigrationHistoryList(ctx context.Context, dbType db.Type, driver db.Driver, find *db.MigrationHistoryFind, baseQuery string) ([]*db.MigrationHistory, error) {
	sqldb, err := driver.GetDbConnection(ctx, bytebaseDatabase)
	if err != nil {
		return nil, err
	}
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queryParams := &db.QueryParams{DatabaseType: dbType}
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
		queryParams.QueryString() +
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
			&history.ExecutionDuration,
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
