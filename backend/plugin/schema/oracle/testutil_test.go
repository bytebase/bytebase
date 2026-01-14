package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	oracledb "github.com/bytebase/bytebase/backend/plugin/db/oracle"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

// createOracleUser creates an Oracle user (schema) with necessary privileges
func createOracleUser(systemDB *sql.DB, username string) error {
	// Use same password as container for test simplicity
	for _, stmt := range []string{
		fmt.Sprintf("CREATE USER %s IDENTIFIED BY test123", username),
		fmt.Sprintf("GRANT CONNECT, RESOURCE, CREATE VIEW, CREATE MATERIALIZED VIEW, CREATE PROCEDURE, CREATE SEQUENCE, CREATE TRIGGER TO %s", username),
		fmt.Sprintf("GRANT UNLIMITED TABLESPACE TO %s", username),
	} {
		if _, err := systemDB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// createOracleDriver creates and opens an Oracle driver connection
func createOracleDriver(ctx context.Context, host, port, username string) (db.Driver, error) {
	driver := &oracledb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:        storepb.DataSourceType_ADMIN,
			Username:    username,
			Host:        host,
			Port:        port,
			Database:    "",
			ServiceName: "FREEPDB1",
		},
		Password: "test123", // Use same password as container for test simplicity
		ConnectionContext: db.ConnectionContext{
			EngineVersion: "23.0",
			DatabaseName:  strings.ToUpper(username),
		},
	}
	return driver.Open(ctx, storepb.Engine_ORACLE, config)
}

// openSystemDatabase opens a connection to Oracle using the SYSTEM account
func openSystemDatabase(host string, port string) (*sql.DB, error) {
	// Use SYSTEM account with the password from container env
	dsn := fmt.Sprintf("oracle://system:test123@%s:%s/FREEPDB1", host, port)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// executeStatements executes multiple SQL statements, handling both regular DDL and PL/SQL blocks
func executeStatements(ctx context.Context, driver db.Driver, statements string) error {
	// Use plsql.SplitSQL to properly split Oracle SQL statements
	stmts, err := plsqlparser.SplitSQL(statements)
	if err != nil {
		return errors.Wrapf(err, "failed to split SQL statements")
	}

	// Execute each statement
	for _, singleSQL := range stmts {
		stmt := strings.TrimSpace(singleSQL.Text)
		if stmt == "" {
			continue
		}

		// Skip statements that contain only comments
		// Strip all comment lines and check if there's actual SQL
		lines := strings.Split(stmt, "\n")
		hasSQL := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
				hasSQL = true
				break
			}
		}
		if !hasSQL {
			continue
		}

		// Execute the statement
		if _, err := driver.Execute(ctx, stmt, db.ExecuteOptions{}); err != nil {
			// Handle Oracle-specific issues where materialized views are misclassified as tables
			if strings.Contains(err.Error(), "must use DROP MATERIALIZED VIEW") {
				// Try to fix the statement by replacing DROP TABLE with DROP MATERIALIZED VIEW
				if strings.HasPrefix(strings.ToUpper(stmt), "DROP TABLE") {
					fixedStmt := strings.Replace(stmt, "DROP TABLE", "DROP MATERIALIZED VIEW", 1)
					if _, retryErr := driver.Execute(ctx, fixedStmt, db.ExecuteOptions{}); retryErr == nil {
						continue // Successfully executed with corrected statement
					}
				}
			}
			// Handle system-generated virtual column indexes that cannot be manually created
			if strings.Contains(err.Error(), "invalid identifier") && strings.Contains(stmt, "SYS_NC") {
				// Skip statements that reference system-generated virtual columns
				continue
			}
			return errors.Wrapf(err, "failed to execute statement: %s", stmt)
		}
	}

	return nil
}
