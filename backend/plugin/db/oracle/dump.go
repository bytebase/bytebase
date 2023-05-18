package oracle

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
	query := fmt.Sprintf(`
select dbms_metadata.get_ddl(object_type, object_name, owner)
from
(
    --Convert DBA_OBJECTS.OBJECT_TYPE to DBMS_METADATA object type:
    select
        owner,
        --Java object names may need to be converted with DBMS_JAVA.LONGNAME.
        --That code is not included since many database don't have Java installed.
        object_name,
        decode(object_type,
            'DATABASE LINK',      'DB_LINK',
            'JOB',                'PROCOBJ',
            'RULE SET',           'PROCOBJ',
            'RULE',               'PROCOBJ',
            'EVALUATION CONTEXT', 'PROCOBJ',
            'CREDENTIAL',         'PROCOBJ',
            'CHAIN',              'PROCOBJ',
            'PROGRAM',            'PROCOBJ',
            'PACKAGE',            'PACKAGE_SPEC',
            'PACKAGE BODY',       'PACKAGE_BODY',
            'TYPE',               'TYPE_SPEC',
            'TYPE BODY',          'TYPE_BODY',
            'MATERIALIZED VIEW',  'MATERIALIZED_VIEW',
            'QUEUE',              'AQ_QUEUE',
            'JAVA CLASS',         'JAVA_CLASS',
            'JAVA TYPE',          'JAVA_TYPE',
            'JAVA SOURCE',        'JAVA_SOURCE',
            'JAVA RESOURCE',      'JAVA_RESOURCE',
            'XML SCHEMA',         'XMLSCHEMA',
            object_type
        ) object_type
    from dba_objects 
    where owner in (%s)
        --These objects are included with other object types.
        and object_type not in ('INDEX PARTITION','INDEX SUBPARTITION',
           'LOB','LOB PARTITION','TABLE PARTITION','TABLE SUBPARTITION')
        --Ignore system-generated types that support collection processing.
        and not (object_type = 'TYPE' and object_name like 'SYS_PLSQL_%%')
        --Exclude nested tables, their DDL is part of their parent table.
        and (owner, object_name) not in (select owner, table_name from dba_nested_tables)
        --Exclude overflow segments, their DDL is part of their parent table.
        and (owner, object_name) not in (select owner, table_name from dba_tables where iot_type = 'IOT_OVERFLOW')
)
order by owner, object_type, object_name
	`, strings.Join(schemas, ","))

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
