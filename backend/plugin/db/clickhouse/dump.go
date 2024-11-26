package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump and restore.
const (
	databaseHeaderFmt = "" +
		"--\n" +
		"-- ClickHouse database structure for `%s`\n" +
		"--\n"
	useDatabaseFmt = "USE `%s`;\n\n"
	tableStmtFmt   = "" +
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
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	if err := dumpTxn(ctx, txn, driver.databaseName, out); err != nil {
		return err
	}

	return txn.Commit()
}

// getDatabases gets all databases of an instance.
func getDatabases(ctx context.Context, txn *sql.Tx) ([]string, error) {
	var dbNames []string
	rows, err := txn.QueryContext(ctx, "SELECT name FROM system.databases")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbNames = append(dbNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dbNames, nil
}

// dumpTxn will dump the input database. schemaOnly isn't supported yet and true by default.
func dumpTxn(ctx context.Context, txn *sql.Tx, database string, out io.Writer) error {
	// Find all dumpable databases
	dbNames, err := getDatabases(ctx, txn)
	if err != nil {
		return errors.Wrap(err, "failed to get databases")
	}

	var dumpableDbNames []string
	if database != "" {
		exist := false
		for _, n := range dbNames {
			if n == database {
				exist = true
				break
			}
		}
		if !exist {
			return common.Errorf(common.NotFound, "database %s not found", database)
		}
		dumpableDbNames = []string{database}
	} else {
		for _, dbName := range dbNames {
			if systemDatabases[dbName] {
				continue
			}
			dumpableDbNames = append(dumpableDbNames, dbName)
		}
	}

	for _, dbName := range dumpableDbNames {
		// Include "USE DATABASE xxx" if dumping multiple databases.
		if len(dumpableDbNames) > 1 {
			// Database header.
			header := fmt.Sprintf(databaseHeaderFmt, dbName)
			if _, err := io.WriteString(out, header); err != nil {
				return err
			}
			dbStmt, err := getDatabaseStmt(ctx, txn, dbName)
			if err != nil {
				return errors.Wrapf(err, "failed to get database %q", dbName)
			}
			if _, err := io.WriteString(out, dbStmt); err != nil {
				return err
			}
			// Use database statement.
			useStmt := fmt.Sprintf(useDatabaseFmt, dbName)
			if _, err := io.WriteString(out, useStmt); err != nil {
				return err
			}
		}

		// Table and view statement.
		tables, err := getTables(ctx, txn, dbName)
		if err != nil {
			return errors.Wrapf(err, "failed to get tables of database %q", dbName)
		}
		for _, tbl := range tables {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tbl.statement)); err != nil {
				return err
			}
		}
	}

	return nil
}

// getDatabaseStmt gets the create statement of a database.
func getDatabaseStmt(ctx context.Context, txn *sql.Tx, dbName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE DATABASE IF NOT EXISTS %s;", dbName)
	var stmt, unused string
	if err := txn.QueryRowContext(ctx, query).Scan(&unused, &stmt); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", err
	}
	return fmt.Sprintf("%s;\n", stmt), nil
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
	query := fmt.Sprintf("SELECT name, engine, create_table_query FROM system.tables WHERE database='%s';", dbName)
	rows, err := txn.QueryContext(ctx, query)
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
