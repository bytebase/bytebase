package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var version, fullVersion string
	if err := driver.db.QueryRowContext(ctx, "SELECT SERVERPROPERTY('productversion'), @@VERSION").Scan(&version, &fullVersion); err != nil {
		return nil, err
	}
	tokens := strings.Fields(fullVersion)
	for _, token := range tokens {
		if len(token) == 4 && strings.HasPrefix(token, "20") {
			version = fmt.Sprintf("%s (%s)", version, token)
			break
		}
	}

	var databases []*storepb.DatabaseSchemaMetadata
	rows, err := driver.db.QueryContext(ctx, "SELECT name, collation_name FROM master.sys.databases WHERE name NOT IN ('master', 'model', 'msdb', 'tempdb', 'rdscore')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseSchemaMetadata{}
		var collation sql.NullString
		if err := rows.Scan(&database.Name, &collation); err != nil {
			return nil, err
		}
		if collation.Valid {
			database.Collation = collation.String
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaNames, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", driver.databaseName)
	}
	tableMap, err := getTables(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
	}
	viewMap, err := getViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name: driver.databaseName,
	}
	for _, schemaName := range schemaNames {
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tableMap[schemaName],
			Views:  viewMap[schemaName],
		})
	}
	return databaseMetadata, nil
}

func getSchemas(txn *sql.Tx) ([]string, error) {
	query := `
		SELECT schema_name FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE schema_name NOT IN ('db_owner', 'db_accessadmin', 'db_backupoperator', 'db_datareader', 'db_datawriter', 'db_ddladmin', 'db_denydatareader', 'db_denydatawriter', 'db_securityadmin', 'guest', 'INFORMATION_SCHEMA', 'sys');`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, err
		}
		result = append(result, schemaName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx) (map[string][]*storepb.TableMetadata, error) {
	columnMap, err := getTableColumns(txn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get table columns")
	}
	indexMap, err := getIndexes(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}
	// TODO(d): foreign keys.
	tableMap := make(map[string][]*storepb.TableMetadata)
	query := `
		SELECT
			SCHEMA_NAME(t.schema_id),
			t.name,
			SUM(ps.row_count)
		FROM sys.tables t
		INNER JOIN sys.dm_db_partition_stats ps ON ps.object_id = t.object_id WHERE index_id < 2
		GROUP BY t.name, t.schema_id
		ORDER BY 1, 2 ASC
		OPTION (RECOMPILE);`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		if err := rows.Scan(&schemaName, &table.Name, &table.RowCount); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tableMap, nil
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	query := `
		SELECT
			table_schema,
			table_name,
			column_name,
			data_type,
			character_maximum_length,
			ordinal_position,
			column_default,
			is_nullable,
			collation_name
		FROM INFORMATION_SCHEMA.COLUMNS
		ORDER BY table_schema, table_name, ordinal_position;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, columnType, nullable string
		var defaultStr, collation sql.NullString
		var characterLength sql.NullInt32
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &columnType, &characterLength, &column.Position, &defaultStr, &nullable, &collation); err != nil {
			return nil, err
		}
		if characterLength.Valid {
			column.Type = fmt.Sprintf("%s(%d)", columnType, characterLength.Int32)
		} else {
			column.Type = columnType
		}
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		column.Collation = collation.String

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	// MSSQL doesn't support function-based indexes.
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)

	query := `
		SELECT
			s.name,
			t.name,
			ind.name,
			col.name,
			ind.type_desc,
			ind.is_primary_key,
			ind.is_unique
		FROM
			sys.indexes ind
		INNER JOIN
			sys.index_columns ic ON  ind.object_id = ic.object_id and ind.index_id = ic.index_id
		INNER JOIN
			sys.columns col ON ic.object_id = col.object_id and ic.column_id = col.column_id
		INNER JOIN
			sys.tables t ON ind.object_id = t.object_id
		INNER JOIN
			sys.schemas s ON s.schema_id = t.schema_id
		WHERE
			t.is_ms_shipped = 0
		ORDER BY 
			s.name, t.name, ind.name, ic.key_ordinal;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, indexName, indexType, colName string
		var primary, unique bool
		if err := rows.Scan(&schemaName, &tableName, &indexName, &colName, &indexType, &primary, &unique); err != nil {
			return nil, err
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*storepb.IndexMetadata)
		}
		if _, ok := indexMap[key][indexName]; !ok {
			indexMap[key][indexName] = &storepb.IndexMetadata{
				Name:    indexName,
				Type:    indexType,
				Unique:  unique,
				Primary: primary,
			}
		}
		indexMap[key][indexName].Expressions = append(indexMap[key][indexName].Expressions, colName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tableIndexes := make(map[db.TableKey][]*storepb.IndexMetadata)
	for k, m := range indexMap {
		for _, v := range m {
			tableIndexes[k] = append(tableIndexes[k], v)
		}
		sort.Slice(tableIndexes[k], func(i, j int) bool {
			return tableIndexes[k][i].Name < tableIndexes[k][j].Name
		})
	}
	return tableIndexes, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := `
		SELECT
			SCHEMA_NAME(v.schema_id) AS schema_name,
			v.name AS view_name,
			m.definition
		FROM sys.views v
		INNER JOIN sys.sql_modules m ON v.object_id = m.object_id
		ORDER BY schema_name, view_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		if err := rows.Scan(&schemaName, &view.Name, &view.Definition); err != nil {
			return nil, err
		}
		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return viewMap, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
