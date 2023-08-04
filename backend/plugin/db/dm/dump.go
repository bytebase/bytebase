package dm

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ bool) (string, error) {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return "", err
	}
	defer txn.Rollback()

	schemas, err := getSchemas(txn)
	if err != nil {
		return "", err
	}

	if len(schemas) == 0 {
		return "", nil
	}

	var quotedSchemas []string
	for _, schema := range schemas {
		quotedSchemas = append(quotedSchemas, fmt.Sprintf("'%s'", schema))
	}
	if err := dumpTxn(ctx, txn, quotedSchemas, out); err != nil {
		return "", err
	}

	if err := txn.Commit(); err != nil {
		return "", err
	}
	return "", nil
}

func dumpTxn(ctx context.Context, txn *sql.Tx, schemas []string, out io.Writer) error {
	// Exclude nested tables, their DDL is part of their parent table.
	// Exclude overflow segments, their DDL is part of their parent table.
	query := fmt.Sprintf(`
		SELECT
			DBMS_METADATA.GET_DDL(u.OBJECT_TYPE, u.OBJECT_NAME, u.OWNER)
		FROM DBA_OBJECTS u
		WHERE
			OWNER IN (%s)
			AND
			u.OBJECT_TYPE IN ('TABLE','INDEX','SEQUENCE','DIRECTORY','VIEW','FUNCTION','PROCEDURE','TABLE PARTITION','INDEX PARTITION','TRIGGER','SCHEDULE','JOB','QUEUE','WINDOW');`,
		strings.Join(schemas, ","))

	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var ddls []string
	for rows.Next() {
		var databaseDDL string
		if err := rows.Scan(&databaseDDL); err != nil {
			return err
		}
		ddls = append(ddls, databaseDDL)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, ddl := range ddls {
		if _, err := io.WriteString(out, ddl); err != nil {
			return err
		}
		if _, err := io.WriteString(out, ";\n"); err != nil {
			return err
		}
	}
	return err
}

// Restore restores a database.
func (*Driver) Restore(_ context.Context, _ io.Reader) (err error) {
	// TODO(d): implement it.
	return nil
}
