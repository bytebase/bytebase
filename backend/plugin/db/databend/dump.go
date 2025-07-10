// Package databend is the plugin for Databend driver.
package databend

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
		"-- Databend database structure for `%s`\n" +
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

	var dumpableDBNames []string
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
		dumpableDBNames = []string{database}
	} else {
		for _, dbName := range dbNames {
			if systemDatabases[dbName] {
				continue
			}
			dumpableDBNames = append(dumpableDBNames, dbName)
		}
	}

	for _, dbName := range dumpableDBNames {
		if len(dumpableDBNames) > 1 {
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
	query := fmt.Sprintf("SHOW CREATE DATABASE %s;", dbName)
	var databaseName, createStmt string
	if err := txn.QueryRowContext(ctx, query).Scan(&databaseName, &createStmt); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", err
	}
	return fmt.Sprintf("%s;\n", createStmt), nil
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
	query := fmt.Sprintf("SELECT name, table_type FROM system.tables WHERE database='%s';", dbName)
	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, tbl := range tables {
		// Databend for table and view both use SHOW CREATE TABLE
		createStmt, err := getCreateStatement(ctx, txn, dbName, tbl.name)
		if err != nil {
			return nil, fmt.Errorf("failed to get create statement for %s.%s: %w", dbName, tbl.name, err)
		}

		createStmt = strings.ReplaceAll(createStmt, fmt.Sprintf("`%s`.", dbName), "")
		createStmt = strings.ReplaceAll(createStmt, fmt.Sprintf(" %s.%s ", dbName, tbl.name), fmt.Sprintf(" %s ", tbl.name))
		tbl.statement = createStmt
	}

	return tables, nil
}

func getCreateStatement(ctx context.Context, txn *sql.Tx, dbName, tableName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE TABLE %s.%s;", dbName, tableName)
	var tableNameResult, createStmt string
	if err := txn.QueryRowContext(ctx, query).Scan(&tableNameResult, &createStmt); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("table/view %s.%s not found", dbName, tableName)
		}
		return "", err
	}
	return fmt.Sprintf("%s;\n", createStmt), nil
}
