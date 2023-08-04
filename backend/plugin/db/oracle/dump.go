package oracle

import (
	"context"
	"database/sql"
	"io"

	"go.uber.org/zap"

	"github.com/pkg/errors"
	go_ora "github.com/sijms/go-ora/v2"

	"github.com/bytebase/bytebase/backend/common/log"
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

	var list []string
	if driver.schemaTenantMode {
		list = append(list, driver.databaseName)
	} else {
		list = append(list, schemas...)
	}
	if err := dumpTxn(ctx, txn, list, out); err != nil {
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
	query := `
		DECLARE
			TYPE type_user_name_list IS TABLE OF VARCHAR2(128) INDEX BY BINARY_INTEGER;
			TYPE type_object_meta IS TABLE OF LONG INDEX BY BINARY_INTEGER;
			PROCEDURE fetch_ddl(
				user_names type_user_name_list,
				ddls OUT type_object_meta
			) IS
				idx number := 1;
			BEGIN
				FOR user_name IN user_names.FIRST .. user_names.LAST LOOP
					ddls(idx) := '/* Schema: ' || user_names(user_name) || ' */' || chr(10) || chr(10) ;
					FOR object_meta IN (
						WITH DISALLOW_OBJECTS AS (
							SELECT OWNER, TABLE_NAME FROM DBA_NESTED_TABLES
							UNION ALL
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
							WHERE OWNER = user_names(user_name) AND U.OBJECT_TYPE IN ('TABLE','INDEX','SEQUENCE','DIRECTORY','VIEW','FUNCTION','PROCEDURE','TRIGGER','SCHEDULE','JOB','QUEUE','WINDOW')
							MINUS
							SELECT OWNER, TABLE_NAME, 'TABLE' FROM DISALLOW_OBJECTS
						)
						SELECT
							U.OBJECT_TYPE, U.OBJECT_NAME, U.OWNER
						FROM NEED_OBJECTS U
					) LOOP
						BEGIN
							ddls(idx) := ddls(idx) || DBMS_METADATA.GET_DDL(object_meta.OBJECT_TYPE, object_meta.OBJECT_NAME, object_meta.OWNER) || ';' || chr(10) || chr(10);
						EXCEPTION
							WHEN OTHERS THEN
							ddls(idx) := ddls(idx) || '/* Error: failed to get ddl for ' || object_meta.OBJECT_TYPE || ' ' || object_meta.OBJECT_NAME || ' in ' || object_meta.OWNER || ' */' || chr(10) || chr(10);
						END;
					END LOOP;
					idx := idx + 1;
				END LOOP;
			END;
		BEGIN
			fetch_ddl(:1, :2);
		END;
	`

	var text []string
	log.Debug("start dumping Oracle schemas", zap.String("query", query))
	if _, err := txn.ExecContext(ctx, query, schemas, go_ora.Out{Dest: &text, Size: len(schemas)}); err != nil {
		return errors.Wrap(err, "failed to dump schemas")
	}
	for _, data := range text {
		if _, err := io.WriteString(out, data); err != nil {
			log.Warn("write error", zap.Error(err))
			return err
		}
		if _, err := io.WriteString(out, "\n"); err != nil {
			log.Warn("write error", zap.Error(err))
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
