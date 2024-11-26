package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump and restore.
const (
	databaseHeaderFmt = "" +
		"--\n" +
		"-- Snowflake database structure for %s\n" +
		"--\n"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := dumpTxn(ctx, txn, driver.databaseName, out); err != nil {
		return err
	}

	err = txn.Commit()
	return err
}

// dumpTxn will dump the input database. schemaOnly isn't supported yet and true by default.
func dumpTxn(ctx context.Context, txn *sql.Tx, database string, out io.Writer) error {
	// Find all dumpable databases
	var dumpableDbNames []string
	if database != "" {
		dumpableDbNames = []string{database}
	} else {
		var err error
		dumpableDbNames, err = getDatabasesTxn(ctx, txn)
		if err != nil {
			return errors.Wrap(err, "failed to get databases")
		}
	}

	for _, dbName := range dumpableDbNames {
		// includeCreateDatabaseStmt should be false if dumping a single database.
		dumpSingleDatabase := len(dumpableDbNames) == 1
		if err := dumpOneDatabase(ctx, txn, dbName, out, dumpSingleDatabase); err != nil {
			return err
		}
	}

	return nil
}

// dumpOneDatabase will dump the database DDL schema for a database.
// Note: this operation is not supported on shared databases, e.g. SNOWFLAKE_SAMPLE_DATA.
func dumpOneDatabase(ctx context.Context, txn *sql.Tx, database string, out io.Writer, dumpSingleDatabase bool) error {
	if !dumpSingleDatabase {
		// Database header.
		header := fmt.Sprintf(databaseHeaderFmt, database)
		if _, err := io.WriteString(out, header); err != nil {
			return err
		}
	}

	query := fmt.Sprintf(`SELECT GET_DDL('DATABASE', '"%s"', true)`, database)
	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databaseDDL string
	for rows.Next() {
		if err := rows.Scan(
			&databaseDDL,
		); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Transform1: if dumpSingleDatabase, we should remove `create or replace database` statement.
	if dumpSingleDatabase {
		lines := strings.Split(databaseDDL, "\n")
		if len(lines) >= 2 {
			lines = lines[2:]
		}
		databaseDDL = strings.Join(lines, "\n")
	}

	// Transform2: remove "create or replace schema PUBLIC;\n\n" because it's created by default.
	schemaStmt := fmt.Sprintf("create or replace schema %s.PUBLIC;", database)
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt+"\n\n", "")
	// If this is the last statement.
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt, "")

	var lines []string
	for _, line := range strings.Split(databaseDDL, "\n") {
		if strings.HasPrefix(strings.ToLower(line), "create ") {
			// Transform3: Remove "DEMO_DB." quantifier.
			line = strings.ReplaceAll(line, fmt.Sprintf(" %s.", database), " ")

			// Transform4 (Important!): replace all `create or replace ` with `create ` to not break existing schema by any chance.
			line = strings.ReplaceAll(line, "create or replace ", "create ")
		}
		lines = append(lines, line)
	}
	databaseDDL = strings.Join(lines, "\n")

	if _, err := io.WriteString(out, databaseDDL); err != nil {
		return err
	}

	return nil
}
