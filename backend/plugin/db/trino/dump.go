package trino

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database schema as SQL statements.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbSchema *storepb.DatabaseSchemaMetadata) error {
	if dbSchema == nil {
		return errors.New("database schema metadata is nil")
	}

	catalog := dbSchema.Name
	if catalog == "" {
		return errors.New("catalog name is empty")
	}

	// Write header
	header := fmt.Sprintf("-- Trino dump of catalog: %s\n", catalog)
	header += fmt.Sprintf("-- Dumped at: %s\n\n", time.Now().Format(time.RFC3339))

	if _, err := io.WriteString(out, header); err != nil {
		return errors.Wrap(err, "failed to write header")
	}

	// Write schema list
	var schemaNames []string
	for _, schema := range dbSchema.Schemas {
		schemaNames = append(schemaNames, schema.Name)
	}
	schemaList := fmt.Sprintf("-- Available schemas in catalog %s:\n", catalog)
	schemaList += fmt.Sprintf("-- %s\n\n", strings.Join(schemaNames, ", "))

	if _, err := io.WriteString(out, schemaList); err != nil {
		return errors.Wrap(err, "failed to write schema list")
	}

	// Process each schema
	for _, schema := range dbSchema.Schemas {
		schemaName := schema.Name

		// Write schema header
		if _, err := io.WriteString(out, fmt.Sprintf("-- Schema: %s\n", schemaName)); err != nil {
			return errors.Wrap(err, "failed to write schema header")
		}

		// Process tables in the schema
		for _, table := range schema.Tables {
			tableName := table.Name

			// Write table header
			tableHeader := fmt.Sprintf("\n-- Table: %s.%s\n", schemaName, tableName)
			if _, err := io.WriteString(out, tableHeader); err != nil {
				return errors.Wrap(err, "failed to write table header")
			}

			// Fetch columns if needed
			if len(table.Columns) == 0 && d.db != nil {
				columns, err := d.fetchTableColumns(ctx, schemaName, tableName)
				if err == nil && len(columns) > 0 {
					table.Columns = columns
				}
			}

			// Write CREATE TABLE statement
			var columnDefs []string
			for _, column := range table.Columns {
				nullable := ""
				if !column.Nullable {
					nullable = " NOT NULL"
				}
				columnDefs = append(columnDefs, fmt.Sprintf("    %s %s%s", column.Name, column.Type, nullable))
			}

			createTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s.%s (\n", catalog, schemaName, tableName)
			if len(columnDefs) > 0 {
				createTable += strings.Join(columnDefs, ",\n")
			}
			createTable += "\n);\n"

			if _, err := io.WriteString(out, createTable); err != nil {
				return errors.Wrap(err, "failed to write create table")
			}
		}
	}

	// Check if anything was found
	if len(dbSchema.Schemas) == 0 {
		if _, err := io.WriteString(out, "-- No schemas found in this catalog\n"); err != nil {
			return errors.Wrap(err, "failed to write no-schemas message")
		}
	} else {
		tablesFound := false
		for _, schema := range dbSchema.Schemas {
			if len(schema.Tables) > 0 {
				tablesFound = true
				break
			}
		}

		if !tablesFound {
			if _, err := io.WriteString(out, "-- No tables found in any schemas\n"); err != nil {
				return errors.Wrap(err, "failed to write no-tables message")
			}
		}
	}

	return nil
}

// Helper function to fetch columns
func (d *Driver) fetchTableColumns(ctx context.Context, schemaName, tableName string) ([]*storepb.ColumnMetadata, error) {
	query := fmt.Sprintf(
		"SELECT column_name, type_name as data_type, 'YES' as is_nullable "+
			"FROM system.jdbc.columns "+
			"WHERE table_schem = '%s' AND table_name = '%s'",
		schemaName, tableName)

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*storepb.ColumnMetadata
	for rows.Next() {
		var name, dataType, isNullable string
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			return nil, err
		}
		columns = append(columns, &storepb.ColumnMetadata{
			Name:     name,
			Type:     dataType,
			Nullable: isNullable == "YES",
		})
	}

	return columns, rows.Err()
}
