// mysqldump is a library for dumping MySQL database schemas provided by bytebase.com.
package mysqldump

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"os"
	"path"

	"github.com/go-sql-driver/mysql"
)

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
)

const (
	useDatabaseFmt = "USE `%s`;\n\n"
	settingsStmt   = "" +
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
	routineStmtFmt = "" +
		"--\n" +
		"-- %s structure for `%s`\n" +
		"--\n" +
		settingsStmt +
		"DELIMITER ;;\n" +
		"%s ;;\n" +
		"DELIMITER ;\n"
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
)

// Dumper is a class for dumping schemas of a MySQL instance.
type Dumper struct {
	db *sql.DB
}

// New creates a new MySQL dumper.
func New(username, password, hostname, port string, tlsCfg *tls.Config) (*Dumper, error) {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/", username, password, hostname, port)
	if tlsCfg != nil {
		mysql.RegisterTLSConfig("custom", tlsCfg)
		dns += "?tls=custom"
	}
	db, err := sql.Open("mysql", dns)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %s", err)
	}
	return &Dumper{
		db: db,
	}, nil
}

// Close closes the dumper.
func (dp *Dumper) Close() error {
	return dp.db.Close()
}

// Dump dumps the schema of a MySQL instance.
func (dp *Dumper) Dump(database, directory string) error {
	dbNames, err := dp.getDatabases()
	if err != nil {
		return fmt.Errorf("failed to get databases: %s", err)
	}

	if database != "" {
		exist := false
		for _, n := range dbNames {
			if n == database {
				exist = true
				break
			}
		}
		if !exist {
			return fmt.Errorf("database %q not found.", database)
		}
		dbNames = []string{database}
	}

	// mysqldump -u root --databases dbName --no-data --routines --events --triggers --compact
	for _, dbName := range dbNames {
		if systemDatabases[dbName] {
			continue
		}

		// Database statement.
		dbStmt, err := dp.getDatabaseStmt(dbName)
		if err != nil {
			return fmt.Errorf("failed to get database %q: %s", dbName, err)
		}
		content := fmt.Sprintf("%s", dbStmt)
		// Use database statement.
		content += fmt.Sprintf(useDatabaseFmt, dbName)

		// Table and view statement.
		tables, err := dp.getTables(dbName)
		if err != nil {
			return fmt.Errorf("failed to get tables of database %q: %s", dbName, err)
		}
		for _, tbl := range tables {
			content += fmt.Sprintf("%s\n", tbl.statement)
		}

		// Procedure and function (routine) statements.
		routines, err := dp.getRoutines(dbName)
		if err != nil {
			return fmt.Errorf("failed to get routines of database %q: %s", dbName, err)
		}
		for _, rt := range routines {
			content += fmt.Sprintf("%s\n", rt.statement)
		}

		// Event statements.
		events, err := dp.getEvents(dbName)
		if err != nil {
			return fmt.Errorf("failed to get events of database %q: %s", dbName, err)
		}
		for _, et := range events {
			content += fmt.Sprintf("%s\n", et.statement)
		}

		// Trigger statements.
		triggers, err := dp.getTriggers(dbName)
		if err != nil {
			return fmt.Errorf("failed to get triggers of database %q: %s", dbName, err)
		}
		for _, tr := range triggers {
			content += fmt.Sprintf("%s\n", tr.statement)
		}

		// Write to files or print.
		if directory == "" {
			fmt.Printf(content)
		} else {
			path := path.Join(directory, fmt.Sprintf("%s.sql", dbName))
			fmt.Printf("database %q: %s\n", dbName, path)
			f, err := os.Create(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := f.WriteString(content); err != nil {
				return err
			}
		}
	}

	return nil
}

// getDatabases gets all databases of an instance.
func (dp *Dumper) getDatabases() ([]string, error) {
	var dbNames []string
	rows, err := dp.db.Query("SHOW DATABASES;")
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
	return dbNames, nil
}

// tableSchema describes the schema of a table or view.
type tableSchema struct {
	name      string
	tableType string
	statement string
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

// getDatabaseStmt gets the create statement of a database.
func (dp *Dumper) getDatabaseStmt(dbName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE DATABASE IF NOT EXISTS %s;", dbName)
	rows, err := dp.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var stmt, unused string
		if err := rows.Scan(&unused, &stmt); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s;\n", stmt), nil
	}
	return "", fmt.Errorf("query %q returned multiple rows.", query)
}

// getTables gets all tables of a database.
func (dp *Dumper) getTables(dbName string) ([]tableSchema, error) {
	var tables []tableSchema
	query := fmt.Sprintf("SHOW FULL TABLES FROM %s;", dbName)
	rows, err := dp.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType); err != nil {
			return nil, err
		}
		stmt, err := dp.getTableStmt(dbName, tbl.name, tbl.tableType)
		if err != nil {
			return nil, fmt.Errorf("getTableStmt(%q, %q, %q) got error: %s", dbName, tbl.name, tbl.tableType, err)
		}
		tbl.statement = stmt
		tables = append(tables, tbl)
	}
	return tables, nil
}

// getTableStmt gets the create statement of a table.
func (dp *Dumper) getTableStmt(dbName, tblName, tblType string) (string, error) {
	switch tblType {
	case "BASE TABLE":
		query := fmt.Sprintf("SHOW CREATE TABLE %s.%s;", dbName, tblName)
		rows, err := dp.db.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		for rows.Next() {
			var stmt, unused string
			if err := rows.Scan(&unused, &stmt); err != nil {
				return "", err
			}
			return fmt.Sprintf(tableStmtFmt, tblName, stmt), nil
		}
		return "", fmt.Errorf("query %q returned invalid rows.", query)
	case "VIEW":
		// This differs from mysqldump as it includes.
		query := fmt.Sprintf("SHOW CREATE VIEW %s.%s;", dbName, tblName)
		rows, err := dp.db.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		for rows.Next() {
			var createStmt, unused string
			if err := rows.Scan(&unused, &createStmt, &unused, &unused); err != nil {
				return "", err
			}
			return fmt.Sprintf(viewStmtFmt, tblName, createStmt), nil
		}
		return "", fmt.Errorf("query %q returned invalid rows.", query)
	default:
		return "", fmt.Errorf("unrecognized table type %q for database %q table %q.", tblType, dbName, tblName)
	}

}

// getRoutines gets all routines of a database.
func (dp *Dumper) getRoutines(dbName string) ([]routineSchema, error) {
	var routines []routineSchema
	for _, routineType := range []string{"FUNCTION", "PROCEDURE"} {
		query := fmt.Sprintf("SHOW %s STATUS WHERE Db = ?;", routineType)
		rows, err := dp.db.Query(query, dbName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		var values []interface{}
		for i := 0; i < len(cols); i++ {
			values = append(values, new(interface{}))
		}
		for rows.Next() {
			var r routineSchema
			if err := rows.Scan(values...); err != nil {
				return nil, err
			}
			r.name = fmt.Sprintf("%s", *values[1].(*interface{}))
			r.routineType = fmt.Sprintf("%s", *values[2].(*interface{}))

			stmt, err := dp.getRoutineStmt(dbName, r.name, r.routineType)
			if err != nil {
				return nil, fmt.Errorf("getRoutineStmt(%q, %q, %q) got error: %s", dbName, r.name, r.routineType, err)
			}
			r.statement = stmt
			routines = append(routines, r)
		}
	}
	return routines, nil
}

// getRoutineStmt gets the create statement of a routine.
func (dp *Dumper) getRoutineStmt(dbName, routineName, routineType string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE %s %s.%s;", routineType, dbName, routineName)
	rows, err := dp.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(routineStmtFmt, getReadableRoutineType(routineType), routineName, charset, charset, collation, sqlmode, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows.", query)

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
func (dp *Dumper) getEvents(dbName string) ([]eventSchema, error) {
	var events []eventSchema
	rows, err := dp.db.Query(fmt.Sprintf("SHOW EVENTS FROM %s;", dbName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []interface{}
	for i := 0; i < len(cols); i++ {
		values = append(values, new(interface{}))
	}
	for rows.Next() {
		var r eventSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		r.name = fmt.Sprintf("%s", *values[1].(*interface{}))
		stmt, err := dp.getEventStmt(dbName, r.name)
		if err != nil {
			return nil, fmt.Errorf("getEventStmt(%q, %q) got error: %s", dbName, r.name, err)
		}
		r.statement = stmt
		events = append(events, r)
	}
	return events, nil
}

// getEventStmt gets the create statement of an event.
func (dp *Dumper) getEventStmt(dbName, eventName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE EVENT %s.%s;", dbName, eventName)
	rows, err := dp.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var sqlmode, timezone, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &timezone, &stmt, &charset, &collation, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(eventStmtFmt, eventName, charset, charset, collation, sqlmode, timezone, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows.", query)
}

// getTriggers gets all triggers of a database.
func (dp *Dumper) getTriggers(dbName string) ([]triggerSchema, error) {
	var triggers []triggerSchema
	rows, err := dp.db.Query(fmt.Sprintf("SHOW TRIGGERS FROM %s;", dbName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var values []interface{}
	for i := 0; i < len(cols); i++ {
		values = append(values, new(interface{}))
	}
	for rows.Next() {
		var tr triggerSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		tr.name = fmt.Sprintf("%s", *values[0].(*interface{}))
		stmt, err := dp.getTriggerStmt(dbName, tr.name)
		if err != nil {
			return nil, fmt.Errorf("getTriggerStmt(%q, %q) got error: %s", dbName, tr.name, err)
		}
		tr.statement = stmt
		triggers = append(triggers, tr)
	}
	return triggers, nil
}

// getTriggerStmt gets the create statement of a trigger.
func (dp *Dumper) getTriggerStmt(dbName, triggerName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE TRIGGER %s.%s;", dbName, triggerName)
	rows, err := dp.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused, &unused); err != nil {
			return "", err
		}
		return fmt.Sprintf(triggerStmtFmt, triggerName, charset, charset, collation, sqlmode, stmt), nil
	}
	return "", fmt.Errorf("query %q returned invalid rows.", query)
}
