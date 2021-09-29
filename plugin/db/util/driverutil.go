package util

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
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
	CheckDuplicateVersion  func(ctx context.Context, tx *sql.Tx, namespace string, engine db.MigrationEngine, version string) (bool, error)
	CheckOutofOrderVersion func(ctx context.Context, tx *sql.Tx, namespace string, engine db.MigrationEngine, version string) (*string, error)
	FindBaseline           func(ctx context.Context, tx *sql.Tx, namespace string) (bool, error)
	FindNextSequence       func(ctx context.Context, tx *sql.Tx, namespace string, requireBaseline bool) (int, error)
	DumpTxn                func(ctx context.Context, txn *sql.Tx, database string, out io.Writer, schemaOnly bool) error
	InsertHistoryQuery     string
	UpdateHistoryQuery     string
}

// ExecuteMigration will execute the database migration.
func ExecuteMigration(ctx context.Context, sqldb *sql.DB, m *db.MigrationInfo, statement string, args MigrationExecutionArgs) (string, error) {
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Phase 1 - Precheck before executing migration
	// Check if the same migration version has alraedy been applied
	duplicate, err := args.CheckDuplicateVersion(ctx, tx, m.Namespace, m.Engine, m.Version)
	if err != nil {
		return "", err
	}
	if duplicate {
		return "", common.Errorf(common.MigrationAlreadyApplied, fmt.Errorf("database %q has already applied version %s", m.Database, m.Version))
	}

	// Check if there is any higher version already been applied
	version, err := args.CheckOutofOrderVersion(ctx, tx, m.Namespace, m.Engine, m.Version)
	if err != nil {
		return "", err
	}
	if version != nil {
		return "", common.Errorf(common.MigrationOutOfOrder, fmt.Errorf("database %q has already applied version %s which is higher than %s", m.Database, *version, m.Version))
	}

	// If the migration engine is VCS and type is not baseline and is not branch, then we can only proceed if there is existing baseline
	// This check is also wrapped in transaction to avoid edge case where two baselinings are running concurrently.
	if m.Engine == db.VCS && m.Type != db.Baseline && m.Type != db.Branch {
		hasBaseline, err := args.FindBaseline(ctx, tx, m.Namespace)
		if err != nil {
			return "", err
		}

		if !hasBaseline {
			return "", common.Errorf(common.MigrationBaselineMissing, fmt.Errorf("%s has not created migration baseline yet", m.Database))
		}
	}

	// VCS based SQL migration requires existing baselining
	requireBaseline := m.Engine == db.VCS && m.Type == db.Migrate
	sequence, err := args.FindNextSequence(ctx, tx, m.Namespace, requireBaseline)
	if err != nil {
		return "", err
	}

	// Phase 2 - Record migration history as PENDING
	// MySQL runs DDL in its own transaction, so we can't commit migration history together with DDL in a single transaction.
	// Thus we sort of doing a 2-phase commit, where we first write a PENDING migration record, and after migration completes, we then
	// update the record to DONE together with the updated schema.
	var schemaBuf bytes.Buffer
	err = args.DumpTxn(ctx, tx, m.Database, &schemaBuf, true /*schemaOnly*/)
	if err != nil {
		return "", formatError(err)
	}
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
		return "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
	}

	insertedId, err := res.LastInsertId()
	if err != nil {
		return "", FormatErrorWithQuery(err, args.InsertHistoryQuery)
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	// Phase 3 - Executing migration
	// Branch migration type always has empty sql.
	// Baseline migration type could also has empty sql when the database is newly created.
	startedTs := time.Now().Unix()
	if statement != "" {
		// MySQL executes DDL in its own transaction, so there is no need to supply a transaction.
		_, err = sqldb.ExecContext(ctx, statement)
		if err != nil {
			return "", formatError(err)
		}
	}
	duration := time.Now().Unix() - startedTs

	afterTx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer afterTx.Rollback()

	// Phase 4 - Dump the schema after migration
	var afterSchemaBuf bytes.Buffer
	err = args.DumpTxn(ctx, afterTx, m.Database, &afterSchemaBuf, true /*schemaOnly*/)
	if err != nil {
		return "", formatError(err)
	}

	// Phase 5 - Update the migration history with 'DONE', execution_duration, updated schema.

	_, err = afterTx.ExecContext(ctx, args.UpdateHistoryQuery,
		duration,
		afterSchemaBuf.String(),
		insertedId,
	)

	if err != nil {
		return "", FormatErrorWithQuery(err, args.UpdateHistoryQuery)
	}

	if err := afterTx.Commit(); err != nil {
		return "", err
	}

	return afterSchemaBuf.String(), nil
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
