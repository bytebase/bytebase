package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
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
func normalizeMetadataForComparison(metadata *storepb.DatabaseSchemaMetadata) {
	// Clear database name as it might differ
	metadata.Name = ""

	// Normalize schemas
	for _, schema := range metadata.Schemas {
		// Normalize tables
		for _, table := range schema.Tables {
			table.DataSize = 0
			table.IndexSize = 0
			table.RowCount = 0

			// Filter out system-generated indexes that reference virtual columns
			var filteredIndexes []*storepb.IndexMetadata
			for _, idx := range table.Indexes {
				// Skip indexes that reference system-generated virtual columns
				hasVirtualColumn := false
				for _, expr := range idx.Expressions {
					if strings.HasPrefix(expr, "SYS_NC") {
						hasVirtualColumn = true
						break
					}
				}
				if !hasVirtualColumn {
					filteredIndexes = append(filteredIndexes, idx)
				}
			}
			table.Indexes = filteredIndexes

			// Sort columns by name for consistent comparison
			sortColumnsByName(table.Columns)

			// Sort indexes by name
			sortIndexesByName(table.Indexes)

			// Sort foreign keys by name
			sortForeignKeysByName(table.ForeignKeys)

			// Sort check constraints by name
			sortCheckConstraintsByName(table.CheckConstraints)
		}

		// Filter out system-generated sequences
		var filteredSequences []*storepb.SequenceMetadata
		for _, seq := range schema.Sequences {
			if !strings.HasPrefix(seq.Name, "ISEQ$$_") {
				filteredSequences = append(filteredSequences, seq)
			}
		}
		schema.Sequences = filteredSequences

		// Sort all collections for consistent comparison
		sortTablesByName(schema.Tables)
		sortViewsByName(schema.Views)
		sortMaterializedViewsByName(schema.MaterializedViews)
		sortFunctionsByName(schema.Functions)
		sortSequencesByName(schema.Sequences)
	}

	// Sort schemas by name
	sortSchemasByName(metadata.Schemas)

	// Sort extensions by name
	sortExtensionsByName(metadata.Extensions)
}

// Sorting helper functions
func sortSchemasByName(schemas []*storepb.SchemaMetadata) {
	slices.SortFunc(schemas, func(a, b *storepb.SchemaMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortTablesByName(tables []*storepb.TableMetadata) {
	slices.SortFunc(tables, func(a, b *storepb.TableMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortColumnsByName(columns []*storepb.ColumnMetadata) {
	slices.SortFunc(columns, func(a, b *storepb.ColumnMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortIndexesByName(indexes []*storepb.IndexMetadata) {
	slices.SortFunc(indexes, func(a, b *storepb.IndexMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortForeignKeysByName(fks []*storepb.ForeignKeyMetadata) {
	slices.SortFunc(fks, func(a, b *storepb.ForeignKeyMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortCheckConstraintsByName(checks []*storepb.CheckConstraintMetadata) {
	slices.SortFunc(checks, func(a, b *storepb.CheckConstraintMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

// normalizeColumnPositions sets all column positions to 0 to ignore position differences during comparison
func normalizeColumnPositions(metadata *storepb.DatabaseSchemaMetadata) {
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			for _, column := range table.Columns {
				column.Position = 0
			}
		}
	}
}

func sortViewsByName(views []*storepb.ViewMetadata) {
	slices.SortFunc(views, func(a, b *storepb.ViewMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortMaterializedViewsByName(mvs []*storepb.MaterializedViewMetadata) {
	slices.SortFunc(mvs, func(a, b *storepb.MaterializedViewMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortFunctionsByName(functions []*storepb.FunctionMetadata) {
	slices.SortFunc(functions, func(a, b *storepb.FunctionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortSequencesByName(sequences []*storepb.SequenceMetadata) {
	slices.SortFunc(sequences, func(a, b *storepb.SequenceMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortExtensionsByName(extensions []*storepb.ExtensionMetadata) {
	slices.SortFunc(extensions, func(a, b *storepb.ExtensionMetadata) int {
		return strings.Compare(a.Name, b.Name)
	})
}
