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

	// Process each schema in the catalog
	for _, schema := range dbSchema.Schemas {
		schemaName := schema.Name

		// Write schema header
		schemaHeader := fmt.Sprintf("-- Schema: %s\n", schemaName)
		if _, err := io.WriteString(out, schemaHeader); err != nil {
			return errors.Wrap(err, "failed to write schema header")
		}

		// We can't create schemas in Trino through SQL in a generic way
		// as it depends on the connector, so we'll just document them

		// Process tables in the schema
		for _, table := range schema.Tables {
			tableName := table.Name

			// Write table header
			tableHeader := fmt.Sprintf("\n-- Table: %s.%s\n", schemaName, tableName)
			if _, err := io.WriteString(out, tableHeader); err != nil {
				return errors.Wrap(err, "failed to write table header")
			}

			// Get column definitions
			var columnDefs []string
			for _, column := range table.Columns {
				nullable := ""
				if !column.Nullable {
					nullable = " NOT NULL"
				}
				columnDef := fmt.Sprintf("    %s %s%s", column.Name, column.Type, nullable)
				columnDefs = append(columnDefs, columnDef)
			}

			// Write CREATE TABLE statement
			// Note: This is just a representation and may not be directly executable
			// depending on the actual connector
			createTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s.%s (\n",
				catalog, schemaName, tableName)
			createTable += strings.Join(columnDefs, ",\n")
			createTable += "\n);\n"

			if _, err := io.WriteString(out, createTable); err != nil {
				return errors.Wrap(err, "failed to write create table")
			}
		}
	}

	return nil
}
