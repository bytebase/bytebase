package trino

import (
	"fmt"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_TRINO, GetDatabaseDefinition)
}

// GetDatabaseDefinition generates the SQL definition for a Trino database schema
func GetDatabaseDefinition(_ schema.GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (string, error) {
	var buf strings.Builder

	// Process each schema
	for _, schema := range metadata.Schemas {
		// Process tables in the schema
		for _, table := range schema.Tables {
			if err := writeCreateTable(&buf, schema.Name, table); err != nil {
				return "", err
			}
			buf.WriteString("\n\n")
		}
	}

	return buf.String(), nil
}

// writeCreateTable generates CREATE TABLE statement
func writeCreateTable(buf *strings.Builder, schema string, table *storepb.TableMetadata) error {
	// Begin CREATE TABLE statement
	createTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\".\"%s\" (\n", schema, table.Name)

	if _, err := buf.WriteString(createTable); err != nil {
		return err
	}

	// Add column definitions
	var columnDefs []string
	for _, column := range table.Columns {
		nullable := ""
		if !column.Nullable {
			nullable = " NOT NULL"
		}
		columnDefs = append(columnDefs, fmt.Sprintf("    \"%s\" %s%s", column.Name, column.Type, nullable))
	}

	if len(columnDefs) > 0 {
		if _, err := buf.WriteString(strings.Join(columnDefs, ",\n")); err != nil {
			return err
		}
	}

	// Close the CREATE TABLE statement
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}

	return nil
}
