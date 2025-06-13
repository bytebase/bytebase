// Package databend is the plugin for Databend driver.
package databend

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	columnQuerytmpl := `
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
		WHERE table_schema = ?
		ORDER BY table_name, ordinal_position`
	columnRows, err := d.db.QueryContext(ctx, columnQuerytmpl, d.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		column := &storepb.ColumnMetadata{}
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
	tableQuerytmpl := `
		SELECT
			name,
			table_type,
			ifNull(table_rows, 0),
			ifNull(data_length, 0),
			table_comment
		FROM information_schema.tables
		WHERE database = %s
		ORDER BY name`
	tableQuery := `
		SELECT
			name,
			table_type,
			ifNull(table_rows, 0),
			ifNull(data_length, 0),
			table_comment
		FROM information_schema.tables
		WHERE database = ?
		ORDER BY name`
	tableRows, err := d.db.QueryContext(ctx, tableQuery, d.databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var name, tableType, comment string
		var rowCount, totalBytes int64
		if err := tableRows.Scan(
			&name,
			&tableType,
			&rowCount,
			&totalBytes,
			&comment,
		); err != nil {
			return nil, err
		}
		txn, err := d.db.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			return nil, err
		}
		definition, err := getCreateStatement(ctx, txn, d.databaseName, name)
		if err != nil {
			return nil, err
		}
		if strings.Contains(strings.ToLower(tableType), "view") {
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
				Engine:   tableType,
				RowCount: rowCount,
				DataSize: totalBytes,
				Comment:  comment,
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
