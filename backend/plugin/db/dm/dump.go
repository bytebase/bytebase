package dm

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
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
	quotedSchemas = append(quotedSchemas, fmt.Sprintf("'%s'", driver.databaseName))
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
		WITH DISALLOW_OBJECTS AS (
			SELECT OWNER, TABLE_NAME FROM DBA_TABLES WHERE IOT_TYPE = 'IOT_OVERFLOW'
		), NEED_OBJECTS AS (
			SELECT
				OWNER,
				OBJECT_NAME,
				decode(object_type,
					'JOB',                'PROCOBJ',
					'QUEUE',              'AQ_QUEUE',
					object_type
				) OBJECT_TYPE
			FROM DBA_OBJECTS U
			WHERE OWNER IN (%s) AND U.OBJECT_TYPE IN ('TABLE','INDEX','SEQUENCE','DIRECTORY','VIEW','FUNCTION','PROCEDURE','TRIGGER','SCHEDULE','JOB','QUEUE','WINDOW')
			MINUS
			SELECT OWNER, TABLE_NAME, 'TABLE' FROM DISALLOW_OBJECTS
		)
		SELECT
			DBMS_METADATA.GET_DDL(U.OBJECT_TYPE, U.OBJECT_NAME, U.OWNER)
		FROM NEED_OBJECTS U`,
		strings.Join(schemas, ","))
	log.Debug("start dumping DM schemas", zap.String("query", query))

	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		log.Warn("query error", zap.Error(err))
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var ddls []string
	for rows.Next() {
		var databaseDDL string
		if err := rows.Scan(&databaseDDL); err != nil {
			log.Warn("ddl scan error", zap.Error(err))
			return err
		}
		ddls = append(ddls, databaseDDL)
	}
	if err := rows.Err(); err != nil {
		log.Warn("rows error", zap.Error(err))
		return err
	}

	for _, ddl := range ddls {
		if _, err := io.WriteString(out, ddl); err != nil {
			log.Warn("write error", zap.Error(err))
			return err
		}
		if _, err := io.WriteString(out, ";\n"); err != nil {
			log.Warn("write newline error", zap.Error(err))
			return err
		}
	}
	return nil
}

// Restore restores a database.
func (*Driver) Restore(_ context.Context, _ io.Reader) (err error) {
	// TODO(d): implement it.
	return nil
}
