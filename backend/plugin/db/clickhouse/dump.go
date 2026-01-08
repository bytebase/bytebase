package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Dump and restore.
const (
	tableStmtFmt = "" +
		"--\n" +
		"-- Table structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	viewStmtFmt = "" +
		"--\n" +
		"-- View structure for `%s`\n" +
		"--\n" +
		"%s;\n"
)

// Dump dumps the database.
func (d *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	txn, err := d.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := dumpTxn(ctx, txn, d.databaseName, out); err != nil {
		return err
	}

	return txn.Commit()
}

// dumpTxn will dump the input database. schemaOnly isn't supported yet and true by default.
func dumpTxn(ctx context.Context, txn *sql.Tx, database string, out io.Writer) error {
	// Table and view statement.
	tables, err := getTables(ctx, txn, database)
	if err != nil {
		return errors.Wrapf(err, "failed to get tables of database %q", database)
	}
	for _, tbl := range tables {
		if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tbl.statement)); err != nil {
			return err
		}
	}

	return nil
}

// tableSchema describes the schema of a table or view.
type tableSchema struct {
	name      string
	tableType string
	statement string
}

// getTables gets all tables of a database.
func getTables(ctx context.Context, txn *sql.Tx, dbName string) ([]*tableSchema, error) {
	var tables []*tableSchema
	query := "SELECT name, engine, create_table_query FROM system.tables WHERE database=?"
	rows, err := txn.QueryContext(ctx, query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType, &tbl.statement); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, tbl := range tables {
		// Remove the database prefix from statement.
		tbl.statement = strings.ReplaceAll(tbl.statement, fmt.Sprintf(" %s.%s ", dbName, tbl.name), fmt.Sprintf(" %s ", tbl.name))
		if tbl.tableType == "View" {
			tbl.statement = fmt.Sprintf(viewStmtFmt, tbl.name, tbl.statement)
		} else {
			tbl.statement = fmt.Sprintf(tableStmtFmt, tbl.name, tbl.statement)
		}
	}
	return tables, nil
}
