package trino

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database schema as SQL statements.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbSchema *storepb.DatabaseSchemaMetadata) error {
	if dbSchema == nil {
		return errors.New("database schema metadata is nil")
	}

	// Populate columns for tables if needed
	for _, s := range dbSchema.Schemas {
		if len(s.Tables) > 0 && d.db != nil {
			columnsMap, err := d.fetchAllColumnsForSchema(ctx, s.Name)
			if err == nil {
				for _, table := range s.Tables {
					if len(table.Columns) == 0 {
						if cols, ok := columnsMap[table.Name]; ok {
							table.Columns = cols
						}
					}
				}
			}
		}
	}

	text, err := schema.GetDatabaseDefinition(storepb.Engine_TRINO, schema.GetDefinitionContext{}, dbSchema)
	if err != nil {
		return errors.Wrapf(err, "failed to get database definition")
	}

	_, err = out.Write([]byte(text))
	return err
}

// fetchAllColumnsForSchema retrieves all column metadata for tables in a schema
func (d *Driver) fetchAllColumnsForSchema(ctx context.Context, schemaName string) (map[string][]*storepb.ColumnMetadata, error) {
	query := fmt.Sprintf(
		"SELECT table_name, column_name, type_name as data_type, is_nullable "+
			"FROM system.jdbc.columns "+
			"WHERE table_schem = '%s'",
		schemaName)

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*storepb.ColumnMetadata)
	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable); err != nil {
			return nil, err
		}

		column := &storepb.ColumnMetadata{
			Name:     columnName,
			Type:     dataType,
			Nullable: isNullable == "YES",
		}

		result[tableName] = append(result[tableName], column)
	}

	return result, rows.Err()
}
