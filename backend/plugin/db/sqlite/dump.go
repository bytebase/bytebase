package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, schemaOnly bool) (string, error) {
	if driver.databaseName == "" {
		return "", errors.Errorf("SQLite can dump one database only at a time")
	}

	// Find all dumpable databases and make sure the existence of the database to be dumped.
	databases, err := driver.getDatabases()
	if err != nil {
		return "", errors.Wrap(err, "failed to get databases")
	}
	exist := false
	for _, n := range databases {
		if n == driver.databaseName {
			exist = true
			break
		}
	}
	if !exist {
		return "", errors.Errorf("database %s not found", driver.databaseName)
	}

	if err := driver.dumpOneDatabase(ctx, out, schemaOnly); err != nil {
		return "", err
	}

	return "", nil
}

type sqliteSchema struct {
	schemaType string
	name       string
	statement  string
}

func (driver *Driver) dumpOneDatabase(ctx context.Context, out io.Writer, schemaOnly bool) error {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	// Get all schemas.
	query := "SELECT type, name, sql FROM sqlite_schema;"
	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var sqliteSchemas []sqliteSchema
	for rows.Next() {
		var s sqliteSchema
		if err := rows.Scan(
			&s.schemaType,
			&s.name,
			&s.statement,
		); err != nil {
			return err
		}
		sqliteSchemas = append(sqliteSchemas, s)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, s := range sqliteSchemas {
		// We should skip sqlite sequence table.
		if s.name == "sqlite_sequence" {
			continue
		}
		if _, err := io.WriteString(out, fmt.Sprintf("%s;\n", s.statement)); err != nil {
			return err
		}

		// Dump table data.
		if !schemaOnly && s.schemaType == "table" {
			if err := exportTableData(txn, s.name, out); err != nil {
				return err
			}
		}
	}

	return txn.Commit()
}

// exportTableData gets the data of a table.
func exportTableData(txn *sql.Tx, tblName string, out io.Writer) error {
	query := fmt.Sprintf("SELECT * FROM `%s`;", tblName)
	rows, err := txn.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return nil
	}
	values := make([]*sql.NullString, len(cols))
	refs := make([]any, len(cols))
	for i := 0; i < len(cols); i++ {
		refs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(refs...); err != nil {
			return err
		}
		tokens := make([]string, len(cols))
		for i, v := range values {
			switch {
			case v == nil || !v.Valid:
				tokens[i] = "NULL"
			default:
				tokens[i] = fmt.Sprintf("'%s'", v.String)
			}
		}
		stmt := fmt.Sprintf("INSERT INTO '%s' VALUES (%s);\n", tblName, strings.Join(tokens, ", "))
		if _, err := io.WriteString(out, stmt); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n"); err != nil {
		return err
	}
	return nil
}

// Restore restores a database.
func (driver *Driver) Restore(ctx context.Context, sc io.Reader) (err error) {
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, sc); err != nil {
		return err
	}
	sqls, err := standard.SplitSQL(buf.String())
	if err != nil {
		return err
	}

	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, sql := range sqls {
		if _, err := txn.Exec(sql.Text); err != nil {
			return err
		}
	}

	return txn.Commit()
}
