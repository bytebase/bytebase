package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	instanceRoles, err := driver.getInstanceRoles(ctx)
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
	rows, err := driver.db.QueryContext(ctx, query)
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
		Metadata: &storepb.InstanceMetadata{
			Roles: instanceRoles,
		},
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
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
	columnRows, err := driver.db.QueryContext(ctx, columnQuery, driver.databaseName)
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
			comment
		FROM system.tables
		WHERE database = $1
		ORDER BY name`
	tableRows, err := driver.db.QueryContext(ctx, tableQuery, driver.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var name, engine, definition, comment string
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
		); err != nil {
			return nil, err
		}
		if engine != "View" {
			schemaMetadata.Tables = append(schemaMetadata.Tables, &storepb.TableMetadata{
				Name:     name,
				Columns:  columnMap[name],
				Engine:   engine,
				RowCount: rowCount,
				DataSize: totalBytes,
				Comment:  comment,
			})
		} else {
			schemaMetadata.Views = append(schemaMetadata.Views, &storepb.ViewMetadata{
				Name:       name,
				Columns:    columnMap[name],
				Definition: definition,
				Comment:    comment,
			})
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	return &storepb.DatabaseSchemaMetadata{
		Name:    driver.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
