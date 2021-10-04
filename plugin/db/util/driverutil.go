package util

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
)

const (
	bytebaseDatabase = "bytebase"
)

// FormatErrorWithQuery will format the error with failed query.
func FormatErrorWithQuery(err error, query string) error {
	return common.Errorf(common.DbExecutionError, fmt.Errorf("failed to execute error: %w\n\nquery:\n%q", err, query))
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
	InsertHistoryQuery string
	UpdateHistoryQuery string
	TablePrefix        string
}

// ExecuteMigration will execute the database migration.
// Returns the created migraiton history id and the updated schema on success.
func ExecuteMigration(ctx context.Context, dbType db.Type, driver db.Driver, m *db.MigrationInfo, statement string, args MigrationExecutionArgs) (int64, string, error) {
	var schemaBuf bytes.Buffer
	// Don't record schema if the database hasn't exist yet.
	if !m.CreateDatabase {
		if err := driver.Dump(ctx, m.Database, &schemaBuf, true /*schemaOnly*/); err != nil {
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
	if version != nil {
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
	insertedId := int64(-1)
	if dbType == db.Postgres {
		tx.QueryRowContext(ctx, args.InsertHistoryQuery,
			m.Creator,
			m.Creator,
			m.Namespace,
			sequence,
			m.Engine,
			m.Type,
			m.Version,
			m.Description,
			statement,
			schemaBuf.String(),
			m.IssueId,
			m.Payload,
		).Scan(&insertedId)
	} else {
		res, err := tx.ExecContext(ctx, args.InsertHistoryQuery,
			m.Creator,
			m.Creator,
			m.Namespace,
			sequence,
			m.Engine,
			m.Type,
			m.Version,
			m.Description,
			statement,
			schemaBuf.String(),
			m.IssueId,
			m.Payload,
		)

		if err != nil {
			return -1, "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
		}

		insertedId, err = res.LastInsertId()
		if err != nil {
			return -1, "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
		}
	}

	if err := tx.Commit(); err != nil {
		return -1, "", err
	}

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could also has empty sql when the database is newly created.
	startedTs := time.Now().Unix()
	if statement != "" {
		// Switch to the database if we're creating a new database
		if !m.CreateDatabase {
			d, err := driver.GetDbConnection(ctx, m.Database)
			if err != nil {
				return -1, "", err
			}
			sqldb = d
		}
		// MySQL executes DDL in its own transaction, so there is no need to supply a transaction.
		_, err = sqldb.ExecContext(ctx, statement)
		if err != nil {
			return -1, "", formatError(err)
		}
	}
	duration := time.Now().Unix() - startedTs

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	if err = driver.Dump(ctx, m.Database, &afterSchemaBuf, true /*schemaOnly*/); err != nil {
		return -1, "", formatError(err)
	}

	// Phase 5 - Update the migration history with 'DONE', execution_duration, updated schema.
	afterSqldb, err := driver.GetDbConnection(ctx, bytebaseDatabase)
	if err != nil {
		return -1, "", err
	}
	afterTx, err := afterSqldb.BeginTx(ctx, nil)
	if err != nil {
		return -1, "", err
	}
	defer afterTx.Rollback()
	_, err = afterTx.ExecContext(ctx, args.UpdateHistoryQuery,
		duration,
		afterSchemaBuf.String(),
		insertedId,
	)

	if err != nil {
		return -1, "", FormatErrorWithQuery(err, args.UpdateHistoryQuery)
	}

	if err := afterTx.Commit(); err != nil {
		return -1, "", err
	}

	return insertedId, afterSchemaBuf.String(), nil
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
	queryParams.AddParam("? < version", fmt.Sprintf("%s", version))
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
func FindMigrationHistoryList(ctx context.Context, dbType db.Type, driver db.Driver, find *db.MigrationHistoryFind, tablePrefix string) ([]*db.MigrationHistory, error) {
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

	var query = `
		SELECT
			id,
			created_by,
			created_ts,
			updated_by,
			updated_ts,
			namespace,
			sequence,
			engine,
			type,
			status,
			version,
			description,
			statement,
			` + `"schema",` + `
			execution_duration,
			issue_id,
			payload
		FROM ` +
		tablePrefix + `migration_history ` +
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
			&history.Namespace,
			&history.Sequence,
			&history.Engine,
			&history.Type,
			&history.Status,
			&history.Version,
			&history.Description,
			&history.Statement,
			&history.Schema,
			&history.ExecutionDuration,
			&history.IssueId,
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
