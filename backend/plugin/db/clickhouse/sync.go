package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	instanceRoles, err := d.getInstanceRoles(ctx)
	if err != nil {
		return nil, err
	}

	var databases []*storepb.DatabaseSchemaMetadata
	// Query db info
	where := fmt.Sprintf("schema_name NOT IN (%s)", systemDatabaseClause)
	query := `
		SELECT
			schema_name
		FROM information_schema.schemata
		WHERE ` + where
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		database := &storepb.DatabaseSchemaMetadata{}
		if err := rows.Scan(
			&database.Name,
		); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
		Metadata: &storepb.Instance{
			Roles: instanceRoles,
		},
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	// Query column info
	// tableName -> columnList map
	columnMap := make(map[string][]*storepb.ColumnMetadata)
	columnQuery := `
		SELECT
			table_name,
			column_name,
			ordinal_position,
			column_default,
			is_nullable,
			column_type,
			ifNull(character_set_name, ''),
			ifNull(collation_name, ''),
			column_comment
		FROM information_schema.columns
		WHERE table_schema = $1
		ORDER BY table_name, ordinal_position`
	columnRows, err := d.db.QueryContext(ctx, columnQuery, d.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		column := &storepb.ColumnMetadata{}
		// Reference: https://clickhouse.com/docs/en/operations/system-tables/information_schema#columns
		// defaultValueExpression is an expression for the default value, or an empty string if it is not defined.
		var tableName, nullable, defaultValueExpression string
		if err := columnRows.Scan(
			&tableName,
			&column.Name,
			&column.Position,
			&defaultValueExpression,
			&nullable,
			&column.Type,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
		); err != nil {
			return nil, err
		}
		nullableBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = nullableBool
		if defaultValueExpression == "" {
			if nullableBool {
				column.DefaultValue = &storepb.ColumnMetadata_DefaultNull{
					DefaultNull: true,
				}
			}
		} else {
			column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{
				DefaultExpression: defaultValueExpression,
			}
		}
		columnMap[tableName] = append(columnMap[tableName], column)
	}
	if err := columnRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	// Query table info
	// We still use system.tables because information_schema.tables doesn't have engine attribute.
	tableQuery := `
		SELECT
			name,
			engine,
			ifNull(total_rows, 0),
			ifNull(total_bytes, 0),
			metadata_modification_time,
			create_table_query,
			comment,
			sorting_key,
			primary_key
		FROM system.tables
		WHERE database = $1
		ORDER BY name`
	tableRows, err := d.db.QueryContext(ctx, tableQuery, d.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var name, engine, definition, comment, sortingKey, primaryKey string
		var rowCount, totalBytes int64
		var lastUpdatedTime time.Time
		if err := tableRows.Scan(
			&name,
			&engine,
			&rowCount,
			&totalBytes,
			&lastUpdatedTime,
			&definition,
			&comment,
			&sortingKey,
			&primaryKey,
		); err != nil {
			return nil, err
		}
		// For view, the engine is "View".
		if engine == "View" {
			schemaMetadata.Views = append(schemaMetadata.Views, &storepb.ViewMetadata{
				Name:       name,
				Columns:    columnMap[name],
				Definition: definition,
				Comment:    comment,
			})
		} else {
			table := &storepb.TableMetadata{
				Name:     name,
				Columns:  columnMap[name],
				Engine:   engine,
				RowCount: rowCount,
				DataSize: totalBytes,
				Comment:  comment,
			}
			indexes, err := d.getDataSkippingIndices(ctx, d.databaseName, name)
			if err != nil {
				return nil, err
			}
			table.Indexes = indexes
			if primaryKey != "" {
				primaryKeys := strings.Split(primaryKey, ", ")
				// Clickhouse save primary keys in `system`.`tables` instead of an index.
				// This is a workaround to make it compatible with our metadata design.
				table.Indexes = append(table.Indexes, &storepb.IndexMetadata{
					Primary:     true,
					Expressions: primaryKeys,
				})
			}
			if sortingKey != "" {
				table.SortingKeys = strings.Split(sortingKey, ", ")
			}
			schemaMetadata.Tables = append(schemaMetadata.Tables, table)
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	return &storepb.DatabaseSchemaMetadata{
		Name:    d.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
}

func (d *Driver) getDataSkippingIndices(ctx context.Context, database string, table string) ([]*storepb.IndexMetadata, error) {
	// Select basic fields of the data skipping index.
	// References:
	// * https://clickhouse.com/docs/en/operations/system-tables/data_skipping_indices
	// * https://clickhouse.com/docs/en/optimize/skipping-indexes
	query := `
		SELECT
			name,
			type,
			expr,
			granularity
		FROM system.data_skipping_indices
		WHERE database = $1 AND table = $2
		ORDER BY name`
	rows, err := d.db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var indices []*storepb.IndexMetadata
	for rows.Next() {
		var expr string
		index := &storepb.IndexMetadata{}
		if err := rows.Scan(
			&index.Name,
			&index.Type,
			&expr,
			&index.Granularity,
		); err != nil {
			return nil, err
		}
		index.Expressions = strings.Split(expr, ", ")
		indices = append(indices, index)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return indices, nil
}
