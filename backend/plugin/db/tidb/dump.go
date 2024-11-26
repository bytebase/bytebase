package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump and restore.
const (
	settingsStmt = "" +
		"SET character_set_client  = %s;\n" +
		"SET character_set_results = %s;\n" +
		"SET collation_connection  = %s;\n" +
		"SET sql_mode              = '%s';\n"
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
	sequenceStmtFmt = "" +
		"--\n" +
		"-- Sequence structure for `%s`\n" +
		"--\n" +
		"%s;\n"
	tempViewStmtFmt = "" +
		"--\n" +
		"-- Temporary view structure for `%s`\n" +
		"--\n" +
		"%s\n"
	routineStmtFmt = "" +
		"--\n" +
		"-- %s structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
	nullRoutineStmtFmt = "" +
		"--\n" +
		"-- %s structure for `%s`\n" +
		"-- NULL statement because user has insufficient permissions.\n" +
		"--\n"
	eventStmtFmt = "" +
		"--\n" +
		"-- Event structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"SET time_zone = '%s';\n" +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
	triggerStmtFmt = "" +
		"--\n" +
		"-- Trigger structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"

	disableUniqueAndForeignKeyCheckStmt = "SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;\n"
	restoreUniqueAndForeignKeyCheckStmt = "SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n"
)

var (
	excludeAutoIncrement  = regexp.MustCompile(` AUTO_INCREMENT=\d+`)
	excludeAutoRandomBase = regexp.MustCompile(` AUTO_RANDOM_BASE=\d+`)
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	// mysqldump -u root --databases dbName --no-data --routines --events --triggers --compact

	// We must use the same MySQL connection to lock and unlock tables.
	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	options := sql.TxOptions{}
	// TiDB does not support readonly, so we only set for MySQL and OceanBase.
	if driver.dbType == storepb.Engine_MYSQL || driver.dbType == storepb.Engine_MARIADB || driver.dbType == storepb.Engine_OCEANBASE {
		options.ReadOnly = true
	}

	txn, err := conn.BeginTx(ctx, &options)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	slog.Debug("begin to dump database", slog.String("database", driver.databaseName))
	if err := dumpTxn(txn, driver.dbType, driver.databaseName, out); err != nil {
		return err
	}

	err = txn.Commit()
	return err
}

// FlushTablesWithReadLock runs FLUSH TABLES table1, table2, ... WITH READ LOCK for all the tables in the database.
func FlushTablesWithReadLock(ctx context.Context, dbType storepb.Engine, conn *sql.Conn, database string) error {
	// The lock acquiring could take a long time if there are concurrent exclusive locks on the tables.
	// We ensures that the execution is canceled after 30 seconds, otherwise we may get dead lock and stuck forever.
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	txn, err := conn.BeginTx(ctxWithTimeout, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	tables, err := getTablesTx(txn, dbType, database)
	if err != nil {
		return err
	}

	var tableNames []string
	for _, table := range tables {
		if table.TableType != baseTableType {
			continue
		}
		tableNames = append(tableNames, fmt.Sprintf("`%s`", table.Name))
	}

	if len(tableNames) != 0 {
		flushTableStmt := fmt.Sprintf("FLUSH TABLES %s WITH READ LOCK;", strings.Join(tableNames, ", "))
		if _, err := txn.ExecContext(ctxWithTimeout, flushTableStmt); err != nil {
			return err
		}
	}

	return txn.Commit()
}

func dumpTxn(txn *sql.Tx, dbType storepb.Engine, database string, out io.Writer) error {
	// Disable foreign key check.
	// mysqldump uses the same mechanism. When there is any schema or data dependency, we have to disable
	// the unique and foreign key check so that the restoring will not fail.
	if _, err := io.WriteString(out, disableUniqueAndForeignKeyCheckStmt); err != nil {
		return err
	}

	// Table and view statement.
	// We have to dump the table before views because of the structure dependency.
	tables, err := getTablesTx(txn, dbType, database)
	if err != nil {
		return errors.Wrapf(err, "failed to get tables of database %q", database)
	}
	// Construct temporal views.
	// Create a temporary view with the same name as the view and with columns of
	// the same name in order to satisfy views that depend on this view.
	// This temporary view will be removed when the actual view is created.
	// The properties of each column, are not preserved in this temporary
	// view. They are not necessary because other views only need to reference
	// the column name, thus we generate SELECT 1 AS colName1, 1 AS colName2.
	// This will not be necessary once we can determine dependencies
	// between views and can simply dump them in the appropriate order.
	// https://sourcegraph.com/github.com/mysql/mysql-server/-/blob/client/mysqldump.cc?L2781
	for _, tbl := range tables {
		if tbl.TableType != viewTableType {
			continue
		}
		if tbl.InvalidView != "" {
			// We will write the invalid view error string to schema.
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", fmt.Sprintf(viewStmtFmt, tbl.Name, fmt.Sprintf("-- %s", tbl.InvalidView)))); err != nil {
				return err
			}
		} else {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", getTemporaryView(tbl.Name, tbl.ViewColumns))); err != nil {
				return err
			}
		}
	}
	// Construct tables.
	for _, tbl := range tables {
		if tbl.TableType == viewTableType {
			continue
		}
		tbl.Statement = excludeSchemaAutoValues(tbl.Statement)
		if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tbl.Statement)); err != nil {
			return err
		}
	}
	// Construct final views.
	for _, tbl := range tables {
		if tbl.TableType != viewTableType {
			continue
		}
		// The temporary view just created above were used to satisfy the schema dependency. See comment above.
		// We have to drop the temporary and incorrect view here to recreate the final and correct one.
		if _, err := io.WriteString(out, fmt.Sprintf("DROP VIEW IF EXISTS `%s`;\n", tbl.Name)); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tbl.Statement)); err != nil {
			return err
		}
	}

	// Procedure and function (routine) statements.
	routines, err := getRoutines(txn, dbType, database)
	if err != nil {
		return errors.Wrapf(err, "failed to get routines of database %q", database)
	}
	for _, rt := range routines {
		if _, err := io.WriteString(out, fmt.Sprintf("%s\n", rt.statement)); err != nil {
			return err
		}
	}

	// OceanBase doesn't support "Event Scheduler"
	if dbType != storepb.Engine_OCEANBASE {
		// Event statements.
		events, err := getEvents(txn, database)
		if err != nil {
			return errors.Wrapf(err, "failed to get events of database %q", database)
		}
		for _, et := range events {
			if _, err := io.WriteString(out, fmt.Sprintf("%s\n", et.statement)); err != nil {
				return err
			}
		}
	}

	// Trigger statements.
	triggers, err := getTriggers(txn, dbType, database)
	if err != nil {
		return errors.Wrapf(err, "failed to get triggers of database %q", database)
	}
	for _, tr := range triggers {
		if _, err := io.WriteString(out, fmt.Sprintf("%s\n", tr.statement)); err != nil {
			return err
		}
	}

	// Restore foreign key check.
	if _, err := io.WriteString(out, restoreUniqueAndForeignKeyCheckStmt); err != nil {
		return err
	}

	return nil
}

func getTemporaryView(name string, columns []string) string {
	var parts []string
	for _, col := range columns {
		parts = append(parts, fmt.Sprintf("1 AS `%s`", col))
	}
	stmt := fmt.Sprintf("CREATE VIEW `%s` AS SELECT\n  %s;\n", name, strings.Join(parts, ",\n  "))
	return fmt.Sprintf(tempViewStmtFmt, name, stmt)
}

// excludeSchemaAutoValues excludes
// 1) the starting value of AUTO_INCREMENT if it's a schema only dump.
// https://github.com/bytebase/bytebase/issues/123
// 2) The auto random base in TiDB.
// /*T![auto_rand_base] AUTO_RANDOM_BASE=39456621 */.
func excludeSchemaAutoValues(s string) string {
	s = excludeAutoRandomBase.ReplaceAllString(s, ``)
	return excludeAutoIncrement.ReplaceAllString(s, ``)
}

// TableSchema describes the schema of a table or view.
type TableSchema struct {
	Name        string
	TableType   string
	Statement   string
	ViewColumns []string
	// InvalidView is the error message indicating an invalid view object.
	InvalidView string
}

// routineSchema describes the schema of a function or procedure (routine).
type routineSchema struct {
	name        string
	routineType string
	statement   string
}

// eventSchema describes the schema of an event.
type eventSchema struct {
	name      string
	statement string
}

// triggerSchema describes the schema of a trigger.
type triggerSchema struct {
	name      string
	statement string
}

// getTablesTx gets all tables of a database using the provided transaction.
func getTablesTx(txn *sql.Tx, dbType storepb.Engine, dbName string) ([]*TableSchema, error) {
	var tables []*TableSchema
	query := "SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?;"
	rows, err := txn.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl TableSchema
		if err := rows.Scan(&tbl.Name, &tbl.TableType); err != nil {
			return nil, err
		}
		tables = append(tables, &tbl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, tbl := range tables {
		stmt, err := getTableStmt(txn, dbType, dbName, tbl.Name, tbl.TableType)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call getTableStmt(%q, %q, %q)", dbName, tbl.Name, tbl.TableType)
		}
		tbl.Statement = stmt
		if tbl.TableType == viewTableType {
			viewColumns, err := getViewColumns(txn, dbName, tbl.Name)
			if err != nil {
				tbl.InvalidView = err.Error()
			} else {
				tbl.ViewColumns = viewColumns
			}
		}
	}
	return tables, nil
}

func trimAfterLastParenthesis(sql string) string {
	pos := strings.LastIndex(sql, ")")
	if pos != -1 {
		return sql[:pos+1]
	}

	return sql
}

// getTableStmt gets the create statement of a table.
func getTableStmt(txn *sql.Tx, dbType storepb.Engine, dbName, tblName, tblType string) (string, error) {
	switch tblType {
	case baseTableType:
		query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`;", dbName, tblName)
		var stmt, unused string
		if err := txn.QueryRow(query).Scan(&unused, &stmt); err != nil {
			if err == sql.ErrNoRows {
				return "", common.FormatDBErrorEmptyRowWithQuery(query)
			}
			return "", err
		}
		if dbType == storepb.Engine_OCEANBASE {
			stmt = trimAfterLastParenthesis(stmt)
		}
		return fmt.Sprintf(tableStmtFmt, tblName, stmt), nil
	case viewTableType:
		// This differs from mysqldump as it includes.
		query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`;", dbName, tblName)
		var createStmt, unused string
		if err := txn.QueryRow(query).Scan(&unused, &createStmt, &unused, &unused); err != nil {
			if err == sql.ErrNoRows {
				return "", common.FormatDBErrorEmptyRowWithQuery(query)
			}
			return "", err
		}
		return fmt.Sprintf(viewStmtFmt, tblName, createStmt), nil
	case sequenceTableType:
		query := fmt.Sprintf("SHOW CREATE SEQUENCE `%s`.`%s`;", dbName, tblName)
		var stmt, unused string
		if err := txn.QueryRow(query).Scan(&unused, &stmt); err != nil {
			if err == sql.ErrNoRows {
				return "", common.FormatDBErrorEmptyRowWithQuery(query)
			}
			return "", err
		}
		return fmt.Sprintf(sequenceStmtFmt, tblName, stmt), nil
	default:
		return "", errors.Errorf("unrecognized table type %q for database %q table %q", tblType, dbName, tblName)
	}
}

// getViewColumns gets the create statement of a table.
func getViewColumns(txn *sql.Tx, dbName, tblName string) ([]string, error) {
	query := fmt.Sprintf("SHOW COLUMNS FROM `%s`.`%s`;", dbName, tblName)
	// https://dev.mysql.com/doc/refman/8.0/en/show-columns.html
	// Field, Type, Null, Key, Default, Extra.
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []string
	for rows.Next() {
		var field string
		var unused sql.NullString
		if err := rows.Scan(&field, &unused, &unused, &unused, &unused, &unused); err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return fields, nil
}

// getRoutines gets all routines of a database.
func getRoutines(txn *sql.Tx, dbType storepb.Engine, dbName string) ([]*routineSchema, error) {
	var routines []*routineSchema
	for _, routineType := range []string{"FUNCTION", "PROCEDURE"} {
		if err := func() error {
			var query string
			if dbType == storepb.Engine_OCEANBASE {
				query = fmt.Sprintf("SHOW %s STATUS FROM `%s`;", routineType, dbName)
			} else {
				query = fmt.Sprintf("SHOW %s STATUS WHERE Db = '%s';", routineType, dbName)
			}
			rows, err := txn.Query(query)
			if err != nil {
				// Oceanbase starts to support functions since 4.0.
				if dbType == storepb.Engine_OCEANBASE {
					return nil
				}
				return errors.Wrapf(err, "failed query %q", query)
			}
			defer rows.Close()

			cols, err := rows.Columns()
			if err != nil {
				return err
			}
			var values []any
			for i := 0; i < len(cols); i++ {
				values = append(values, new(any))
			}
			for rows.Next() {
				var r routineSchema
				if err := rows.Scan(values...); err != nil {
					return err
				}
				r.name = fmt.Sprintf("%s", *values[1].(*any))
				r.routineType = fmt.Sprintf("%s", *values[2].(*any))

				routines = append(routines, &r)
			}
			return rows.Err()
		}(); err != nil {
			return nil, err
		}
	}

	for _, r := range routines {
		stmt, err := getRoutineStmt(txn, dbName, r.name, r.routineType)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call getRoutineStmt(%q, %q, %q)", dbName, r.name, r.routineType)
		}
		r.statement = stmt
	}
	return routines, nil
}

// getRoutineStmt gets the create statement of a routine.
func getRoutineStmt(txn *sql.Tx, dbName, routineName, routineType string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE %s `%s`.`%s`;", routineType, dbName, routineName)
	var sqlmode, charset, collation, unused string
	var stmt sql.NullString
	if err := txn.QueryRow(query).Scan(
		&unused,
		&sqlmode,
		&stmt,
		&charset,
		&collation,
		&unused,
	); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", err
	}
	if !stmt.Valid {
		// https://dev.mysql.com/doc/refman/8.0/en/show-create-procedure.html
		slog.Warn("Statement is null, user does not have sufficient permissions",
			slog.String("routineType", routineType),
			slog.String("dbName", dbName),
			slog.String("routineName", routineName))
		return fmt.Sprintf(nullRoutineStmtFmt, getReadableRoutineType(routineType), routineName), nil
	}
	return fmt.Sprintf(routineStmtFmt, getReadableRoutineType(routineType), routineName, charset, charset, collation, sqlmode, stmt.String), nil
}

// getReadableRoutineType gets the printable routine type.
func getReadableRoutineType(s string) string {
	switch s {
	case "FUNCTION":
		return "Function"
	case "PROCEDURE":
		return "Procedure"
	default:
		return s
	}
}

// getEvents gets all events of a database.
func getEvents(txn *sql.Tx, dbName string) ([]*eventSchema, error) {
	var events []*eventSchema
	query := fmt.Sprintf("SHOW EVENTS FROM `%s`;", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []any
	for i := 0; i < len(cols); i++ {
		values = append(values, new(any))
	}
	for rows.Next() {
		var r eventSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		r.name = fmt.Sprintf("%s", *values[1].(*any))
		events = append(events, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	for _, r := range events {
		stmt, err := getEventStmt(txn, dbName, r.name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call getEventStmt(%q, %q)", dbName, r.name)
		}
		r.statement = stmt
	}
	return events, nil
}

// getEventStmt gets the create statement of an event.
func getEventStmt(txn *sql.Tx, dbName, eventName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE EVENT `%s`.`%s`;", dbName, eventName)
	var sqlmode, timezone, stmt, charset, collation, unused string
	if err := txn.QueryRow(query).Scan(&unused, &sqlmode, &timezone, &stmt, &charset, &collation, &unused); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", err
	}
	return fmt.Sprintf(eventStmtFmt, eventName, charset, charset, collation, sqlmode, timezone, stmt), nil
}

// getTriggers gets all triggers of a database.
func getTriggers(txn *sql.Tx, dbType storepb.Engine, dbName string) ([]*triggerSchema, error) {
	var triggers []*triggerSchema
	query := fmt.Sprintf("SHOW TRIGGERS FROM `%s`;", dbName)
	rows, err := txn.Query(query)
	if err != nil {
		// Oceanbase starts to support trigger since 4.0.
		if dbType == storepb.Engine_OCEANBASE {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []any
	for i := 0; i < len(cols); i++ {
		values = append(values, new(any))
	}
	for rows.Next() {
		var tr triggerSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		tr.name = fmt.Sprintf("%s", *values[0].(*any))
		triggers = append(triggers, &tr)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	for _, tr := range triggers {
		stmt, err := getTriggerStmt(txn, dbName, tr.name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to call getTriggerStmt(%q, %q)", dbName, tr.name)
		}
		tr.statement = stmt
	}
	return triggers, nil
}

// getTriggerStmt gets the create statement of a trigger.
func getTriggerStmt(txn *sql.Tx, dbName, triggerName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE TRIGGER `%s`.`%s`;", dbName, triggerName)
	rows, err := txn.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var sqlmode, stmt, charset, collation string
	var unused any
	for rows.Next() {
		cols := make([]any, len(columns))
		// The query SHOW CREATE TRIGGER returns uncertain number of columns.
		for i := 0; i < len(columns); i++ {
			switch columns[i] {
			case "sql_mode":
				cols[i] = &sqlmode
			case "SQL Original Statement":
				cols[i] = &stmt
			case "character_set_client":
				cols[i] = &charset
			case "collation_connection":
				cols[i] = &collation
			default:
				cols[i] = &unused
			}
		}
		if err := rows.Scan(cols...); err != nil {
			return "", errors.Wrapf(err, "cannot scan row from %q query", query)
		}
	}
	if err := rows.Err(); err != nil {
		return "", util.FormatErrorWithQuery(err, query)
	}
	return fmt.Sprintf(triggerStmtFmt, triggerName, charset, charset, collation, sqlmode, stmt), nil
}
