package dm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const systemSchema = "'CTISYS','SYS','SYSAUDITOR','SYSDBA','SYSSSO'"

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var fullVersion string
	// DM Database Server 64 V8.
	if err := driver.db.QueryRowContext(ctx, "SELECT BANNER FROM v$version WHERE banner LIKE 'DM%'").Scan(&fullVersion); err != nil {
		return nil, err
	}
	tokens := strings.Fields(fullVersion)
	version := tokens[len(tokens)-1]

	databases, err := driver.syncSchemaTenantModeInstance(ctx)
	if err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
	}, nil
}

func (driver *Driver) syncSchemaTenantModeInstance(ctx context.Context) ([]*storepb.DatabaseMetadata, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemas, err := getSchemas(txn)
	if err != nil {
		return nil, err
	}

	var result []*storepb.DatabaseMetadata

	for _, schema := range schemas {
		result = append(result, &storepb.DatabaseMetadata{
			Name:        schema,
			ServiceName: "",
		})
	}

	return result, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseMetadata, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	tableMap, err := getTables(txn, driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
	}
	viewMap, err := getViews(txn, driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseMetadata{
		Name: driver.databaseName,
	}
	databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
		Name:   driver.databaseName,
		Tables: tableMap[driver.databaseName],
		Views:  viewMap[driver.databaseName],
	})
	return databaseMetadata, nil
}

func getSchemas(txn *sql.Tx) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT username FROM all_users
		WHERE username NOT IN (%s) ORDER BY username`,
		systemSchema)
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
func getTables(txn *sql.Tx, schemaName string) (map[string][]*storepb.TableMetadata, error) {
	columnMap, err := getTableColumns(txn, schemaName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get table columns")
	}
	indexMap, err := getIndexes(txn, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}
	// TODO(d): foreign keys.
	tableMap := make(map[string][]*storepb.TableMetadata)
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, NUM_ROWS
		FROM all_tables
		WHERE OWNER NOT IN (%s) 
		ORDER BY OWNER, TABLE_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, NUM_ROWS
		FROM all_tables
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME`, schemaName)
	}

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		var count sql.NullInt64
		if err := rows.Scan(&schemaName, &table.Name, &count); err != nil {
			return nil, err
		}
		table.RowCount = count.Int64
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
func getTableColumns(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	// https://github.com/bytebase/bytebase/issues/6663
	// Invisible columns don't have column ID so that we need to filter out them.
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT
			OWNER,
			TABLE_NAME,
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_ID,
			DATA_DEFAULT,
			NULLABLE
		FROM sys.all_tab_columns
		WHERE OWNER NOT IN (%s) AND COLUMN_ID IS NOT NULL
		ORDER BY OWNER, TABLE_NAME, COLUMN_ID`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT
			OWNER,
			TABLE_NAME,
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_ID,
			DATA_DEFAULT,
			NULLABLE
		FROM sys.all_tab_columns
		WHERE OWNER = '%s' AND COLUMN_ID IS NOT NULL
		ORDER BY TABLE_NAME, COLUMN_ID`, schemaName)
	}

	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, nullable string
		var defaultStr sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &column.Position, &defaultStr, &nullable); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		// TODO(d): add collation.

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	expressionsMap := make(map[db.IndexKey][]string)
	queryColumn := ""
	if schemaName == "" {
		queryColumn = fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_NAME
		FROM sys.all_ind_columns
		WHERE TABLE_OWNER NOT IN (%s) 
		ORDER BY TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, systemSchema)
	} else {
		queryColumn = fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_NAME
		FROM sys.all_ind_columns
		WHERE TABLE_OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, schemaName)
	}
	colRows, err := txn.Query(queryColumn)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()
	for colRows.Next() {
		var schemaName, tableName, indexName, columnName string
		if err := colRows.Scan(&schemaName, &tableName, &indexName, &columnName); err != nil {
			return nil, err
		}
		key := db.IndexKey{Schema: schemaName, Table: tableName, Index: indexName}
		expressionsMap[key] = append(expressionsMap[key], columnName)
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}
	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, INDEX_NAME, UNIQUENESS, INDEX_TYPE
		FROM sys.all_indexes
		WHERE OWNER NOT IN (%s) 
		ORDER BY OWNER, TABLE_NAME, INDEX_NAME`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, INDEX_NAME, UNIQUENESS, INDEX_TYPE
		FROM sys.all_indexes
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME`, schemaName)
	}
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		index := &storepb.IndexMetadata{}
		var schemaName, tableName, unique string
		// INDEX_TYPE is NORMAL, or FUNCTION-BASED NORMAL.
		if err := rows.Scan(&schemaName, &tableName, &index.Name, &unique, &index.Type); err != nil {
			return nil, err
		}

		index.Unique = unique == "UNIQUE"
		indexKey := db.IndexKey{Schema: schemaName, Table: tableName, Index: index.Name}
		index.Expressions = expressionsMap[indexKey]
		if err != nil {
			return nil, err
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexMap, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx, schemaName string) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := ""
	if schemaName == "" {
		query = fmt.Sprintf(`
		SELECT OWNER, VIEW_NAME, TEXT
		FROM sys.all_views
		WHERE OWNER NOT IN (%s) 
		ORDER BY owner, view_name
	`, systemSchema)
	} else {
		query = fmt.Sprintf(`
		SELECT OWNER, VIEW_NAME, TEXT
		FROM sys.all_views
		WHERE OWNER = '%s'
		ORDER BY view_name
	`, schemaName)
	}

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
