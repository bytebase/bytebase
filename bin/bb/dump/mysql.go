// dump is a library for dumping database schemas provided by bytebase.com.
package dump

import (
	"database/sql"
	"fmt"
	"os"
	"path"

	_ "github.com/go-sql-driver/mysql"
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

// tableSchema describes the schema of a table or view.
type tableSchema struct {
	name       string
	tableType  string
	createStmt string
}

// routineSchema describes the schema of a function or procedure (routine).
type routineSchema struct {
	name        string
	routineType string
	createStmt  string
}

// eventSchema describes the schema of an event.
type eventSchema struct {
	name       string
	createStmt string
}

// triggerSchema describes the schema of a trigger.
type triggerSchema struct {
	name       string
	createStmt string
}

// getDatabases gets all databases from an instance.
func getDatabases(db *sql.DB) ([]string, error) {
	var dbNames []string
	rows, err := db.Query("SHOW DATABASES;")
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

// getDatabaseStmt gets the create statement of a database.
func getDatabaseStmt(db *sql.DB, dbName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE DATABASE IF NOT EXISTS %s;", dbName)
	rows, err := db.Query(query)
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
	return "", fmt.Errorf("query %q returned multiple rows", query)
}

// getTables gets all tables from a database.
func getTables(db *sql.DB, dbName string) ([]tableSchema, error) {
	var tables []tableSchema
	query := fmt.Sprintf("SHOW FULL TABLES FROM %s;", dbName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		if err := rows.Scan(&tbl.name, &tbl.tableType); err != nil {
			return nil, err
		}
		if err = getTableStmt(db, dbName, &tbl); err != nil {
			return nil, fmt.Errorf("getTableStmt(%q, %q) got error %v", dbName, tbl.name, err)
		}
		tables = append(tables, tbl)
	}
	return tables, nil
}

// getTableStmt gets the create statement of a table.
func getTableStmt(db *sql.DB, dbName string, tbl *tableSchema) error {
	switch tbl.tableType {
	case "BASE TABLE":
		query := fmt.Sprintf("SHOW CREATE TABLE %s.%s;", dbName, tbl.name)
		rows, err := db.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var stmt, unused string
			if err := rows.Scan(&unused, &stmt); err != nil {
				return err
			}
			tbl.createStmt = fmt.Sprintf(tableStmtFmt, tbl.name, stmt)
			return nil
		}
		return fmt.Errorf("query %q returned multiple rows", query)
	case "VIEW":
		// This differs from mysqldump as it includes.
		query := fmt.Sprintf("SHOW CREATE VIEW %s.%s;", dbName, tbl.name)
		rows, err := db.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var createStmt, unused string
			if err := rows.Scan(&unused, &createStmt, &unused, &unused); err != nil {
				return err
			}
			tbl.createStmt = fmt.Sprintf(viewStmtFmt, tbl.name, createStmt)
			return nil
		}
		return fmt.Errorf("query %q returned multiple rows", query)
	default:
		return fmt.Errorf("unrecognized table type %q for database %q table %q", tbl.tableType, dbName, tbl.name)
	}

}

// getRoutines gets all routines of a database.
func getRoutines(db *sql.DB, dbName string) ([]routineSchema, error) {
	var routines []routineSchema
	for _, routineType := range []string{"FUNCTION", "PROCEDURE"} {
		query := fmt.Sprintf("SHOW %s STATUS WHERE Db = ?;", routineType)
		rows, err := db.Query(query, dbName)
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

			if err := getRoutineStmt(db, dbName, &r); err != nil {
				return nil, fmt.Errorf("getRoutineStmt(%q, %q, %q) got error %v", dbName, r.name, r.routineType, err)
			}
			routines = append(routines, r)
		}
	}
	return routines, nil
}

// getRoutineStmt gets the create statement of a routine.
func getRoutineStmt(db *sql.DB, dbName string, rt *routineSchema) error {
	query := fmt.Sprintf("SHOW CREATE %s %s.%s;", rt.routineType, dbName, rt.name)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused); err != nil {
			return err
		}
		rt.createStmt = fmt.Sprintf(routineStmtFmt, getReadableRoutineType(rt.routineType), rt.name, charset, charset, collation, sqlmode, stmt)
		return nil
	}
	return fmt.Errorf("query %q returned multiple rows", query)

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
func getEvents(db *sql.DB, dbName string) ([]eventSchema, error) {
	var events []eventSchema
	rows, err := db.Query(fmt.Sprintf("SHOW EVENTS FROM %s;", dbName))
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
		if err := getEventStmt(db, dbName, &r); err != nil {
			return nil, fmt.Errorf("getEventStmt(%q, %q) got error %v", dbName, r.name, err)
		}
		events = append(events, r)
	}
	return events, nil
}

// getEventStmt gets the create statement of an event.
func getEventStmt(db *sql.DB, dbName string, event *eventSchema) error {
	query := fmt.Sprintf("SHOW CREATE EVENT %s.%s;", dbName, event.name)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sqlmode, timezone, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &timezone, &stmt, &charset, &collation, &unused); err != nil {
			return err
		}
		event.createStmt = fmt.Sprintf(eventStmtFmt, event.name, charset, charset, collation, sqlmode, timezone, stmt)
		return nil
	}
	return fmt.Errorf("query %q returned multiple rows", query)
}

// getTriggers gets all triggers of a database.
func getTriggers(db *sql.DB, dbName string) ([]triggerSchema, error) {
	var triggers []triggerSchema
	rows, err := db.Query(fmt.Sprintf("SHOW TRIGGERS FROM %s;", dbName))
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
		var r triggerSchema
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		r.name = fmt.Sprintf("%s", *values[0].(*interface{}))
		if err = getTriggerStmt(db, dbName, &r); err != nil {
			return nil, fmt.Errorf("getTriggerStmt(%q, %q) got error %v", dbName, r.name, err)
		}
		triggers = append(triggers, r)
	}
	return triggers, nil
}

// getTriggerStmt gets the create statement of a trigger.
func getTriggerStmt(db *sql.DB, dbName string, tr *triggerSchema) error {
	query := fmt.Sprintf("SHOW CREATE TRIGGER %s.%s;", dbName, tr.name)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sqlmode, stmt, charset, collation, unused string
		if err := rows.Scan(&unused, &sqlmode, &stmt, &charset, &collation, &unused, &unused); err != nil {
			return err
		}
		tr.createStmt = fmt.Sprintf(triggerStmtFmt, tr.name, charset, charset, collation, sqlmode, stmt)
		return nil
	}
	return fmt.Errorf("query %q returned multiple rows", query)
}

// MysqlDump dumps the schema of an instance.
func MysqlDump(username, password, hostname, port, directory string) error {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/", username, password, hostname, port)
	db, err := sql.Open("mysql", dns)
	if err != nil {
		return fmt.Errorf("failed to open database: %s", err)
	}
	defer db.Close()

	dbNames, err := getDatabases(db)
	if err != nil {
		return fmt.Errorf("failed to get databases: %s", err)
	}

	for _, dbName := range dbNames {
		if systemDatabases[dbName] {
			continue
		}

		// Database statement.
		dbStmt, err := getDatabaseStmt(db, dbName)
		if err != nil {
			return fmt.Errorf("failed to get database %q: %s", dbName, err)
		}

		// Database statement.
		content := fmt.Sprintf("%s", dbStmt)

		// Use database statement.
		content += fmt.Sprintf(useDatabaseFmt, dbName)

		// Tables and views statement.
		tables, err := getTables(db, dbName)
		if err != nil {
			return fmt.Errorf("failed to get tables from database %q: %s", dbName, err)
		}
		for _, tbl := range tables {
			content += fmt.Sprintf("%s\n", tbl.createStmt)
		}
		content += "\n"

		// Procedures and functions (routines) statements.
		routines, err := getRoutines(db, dbName)
		if err != nil {
			return fmt.Errorf("failed to get routines from database %q: %s", dbName, err)
		}
		for _, rt := range routines {
			content += fmt.Sprintf("%s\n", rt.createStmt)
		}
		content += "\n"

		// Events statements.
		events, err := getEvents(db, dbName)
		if err != nil {
			return fmt.Errorf("failed to get events from database %q: %s", dbName, err)
		}
		for _, et := range events {
			content += fmt.Sprintf("%s\n", et.createStmt)
		}
		content += "\n"

		// Trigger statements.
		triggers, err := getTriggers(db, dbName)
		if err != nil {
			return fmt.Errorf("failed to get triggers from database %q: %s", dbName, err)
		}
		for _, tr := range triggers {
			content += fmt.Sprintf("%s\n", tr.createStmt)
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
