package mssql

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mssqldb "github.com/bytebase/bytebase/backend/plugin/db/mssql"
)

// createMSSQLDriver creates and opens a MSSQL driver connection
func createMSSQLDriver(ctx context.Context, host, port, database string) (db.Driver, error) {
	driver := &mssqldb.Driver{}
	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "sa",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "Test123!",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: database,
		},
	}
	return driver.Open(ctx, storepb.Engine_MSSQL, config)
}

// executeSQL executes SQL statements, handling GO separators
func executeSQL(ctx context.Context, driver db.Driver, sql string) error {
	if strings.TrimSpace(sql) == "" {
		return nil
	}

	// Split by GO statements (case insensitive)
	statements := splitByGO(sql)

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := driver.Execute(ctx, stmt, db.ExecuteOptions{}); err != nil {
			return errors.Wrapf(err, "failed to execute statement %d: %s", i+1, stmt)
		}
	}

	return nil
}

// splitByGO splits SQL by GO statements (case insensitive) or by semicolons if no GO statements
func splitByGO(sql string) []string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return []string{}
	}

	// First check if there are any GO statements
	hasGOStatements := false
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "GO") {
			hasGOStatements = true
			break
		}
	}

	if hasGOStatements {
		// Split by GO statements
		var statements []string
		var currentStatement strings.Builder

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.EqualFold(trimmed, "GO") {
				if currentStatement.Len() > 0 {
					statements = append(statements, currentStatement.String())
					currentStatement.Reset()
				}
			} else {
				if currentStatement.Len() > 0 {
					currentStatement.WriteString("\n")
				}
				currentStatement.WriteString(line)
			}
		}

		if currentStatement.Len() > 0 {
			statements = append(statements, currentStatement.String())
		}

		return statements
	}
	// Split by semicolons for DDL statements
	statements := strings.Split(sql, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			// Check if this statement contains any non-comment SQL
			lines := strings.Split(stmt, "\n")
			hasSQL := false
			var sqlLines []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "--") {
					hasSQL = true
					sqlLines = append(sqlLines, line)
				}
			}
			// Only include if there's actual SQL (not just comments)
			if hasSQL {
				result = append(result, strings.Join(sqlLines, " "))
			}
		}
	}
	return result
}
