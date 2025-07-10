package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Dump dumps the database.
func (d *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	if d.databaseName == "" {
		return errors.Errorf("SQLite can dump one database only at a time")
	}

	// Find all dumpable databases and make sure the existence of the database to be dumped.
	databases, err := d.getDatabases()
	if err != nil {
		return errors.Wrap(err, "failed to get databases")
	}
	exist := false
	for _, n := range databases {
		if n == d.databaseName {
			exist = true
			break
		}
	}
	if !exist {
		return errors.Errorf("database %s not found", d.databaseName)
	}

	return d.dumpOneDatabase(ctx, out)
}

type sqliteSchema struct {
	schemaType string
	name       string
	statement  string
}

func (d *Driver) dumpOneDatabase(ctx context.Context, out io.Writer) error {
	txn, err := d.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
		var sql sql.NullString
		if err := rows.Scan(
			&s.schemaType,
			&s.name,
			&sql,
		); err != nil {
			return err
		}
		if sql.Valid {
			s.statement = sql.String
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
	}

	return txn.Commit()
}
