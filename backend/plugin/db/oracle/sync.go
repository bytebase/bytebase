package oracle

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var version string
	if err := driver.db.QueryRowContext(ctx, "SELECT BANNER FROM v$version WHERE banner LIKE 'Oracle%'").Scan(&version); err != nil {
		return nil, err
	}

	var databases []*storepb.DatabaseMetadata
	rows, err := driver.db.QueryContext(ctx, "SELECT name FROM v$database")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseMetadata{}
		if err := rows.Scan(&database.Name); err != nil {
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
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context, databaseName string) (*storepb.DatabaseMetadata, error) {
	// TODO(d): filter out system tables.
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaNames, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", databaseName)
	}
	tableMap, err := getTables(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", databaseName)
	}
	viewMap, err := getViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseMetadata{
		Name: databaseName,
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
	query := `SELECT username FROM all_users WHERE ORACLE_MAINTAINED = 'N' AND USERNAME != 'OPS$ORACLE' ORDER BY username`
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
		SELECT OWNER, TABLE_NAME, NUM_ROWS
		FROM all_tables
		WHERE OWNER IN (
			SELECT USERNAME FROM DBA_USERS WHERE ORACLE_MAINTAINED = 'N'
		) AND OWNER != 'OPS$ORACLE'
		ORDER BY OWNER, TABLE_NAME`
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
func getTableColumns(txn *sql.Tx) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	query := `
		SELECT
			OWNER,
			TABLE_NAME,
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_ID,
			DATA_DEFAULT,
			NULLABLE,
			COLLATION
		FROM sys.all_tab_columns
		WHERE OWNER IN (
			SELECT USERNAME FROM DBA_USERS WHERE ORACLE_MAINTAINED = 'N'
		) AND OWNER != 'OPS$ORACLE'
		ORDER BY OWNER, TABLE_NAME, COLUMN_ID`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, nullable string
		var defaultStr, collation sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &column.Position, &defaultStr, &nullable, &collation); err != nil {
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
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	expressionsMap := make(map[db.IndexKey][]string)
	queryColumn := `
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_NAME
		FROM sys.all_ind_columns
		WHERE TABLE_OWNER IN (
			SELECT USERNAME FROM DBA_USERS WHERE ORACLE_MAINTAINED = 'N'
		) AND TABLE_OWNER != 'OPS$ORACLE'
		ORDER BY TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_POSITION`
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
	queryExpression := `
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_EXPRESSION, COLUMN_POSITION
		FROM sys.all_ind_expressions
		WHERE TABLE_OWNER IN (
			SELECT USERNAME FROM DBA_USERS WHERE ORACLE_MAINTAINED = 'N'
		) AND TABLE_OWNER != 'OPS$ORACLE'
		ORDER BY TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_POSITION`
	expRows, err := txn.Query(queryExpression)
	if err != nil {
		return nil, err
	}
	defer expRows.Close()
	for expRows.Next() {
		var schemaName, tableName, indexName, columnExpression string
		var position int
		if err := expRows.Scan(&schemaName, &tableName, &indexName, &columnExpression, &position); err != nil {
			return nil, err
		}
		key := db.IndexKey{Schema: schemaName, Table: tableName, Index: indexName}
		// Position starts from 1.
		expIndex := position - 1
		if expIndex >= len(expressionsMap[key]) {
			return nil, errors.Errorf("expression %q position %v out of range for index %q.%q.%q", columnExpression, position, schemaName, tableName, indexName)
		}
		expressionsMap[key][expIndex] = columnExpression
	}
	if err := expRows.Err(); err != nil {
		return nil, err
	}
	query := `
		SELECT OWNER, TABLE_NAME, INDEX_NAME, UNIQUENESS, INDEX_TYPE
		FROM sys.all_indexes
		WHERE OWNER IN (
			SELECT USERNAME FROM DBA_USERS WHERE ORACLE_MAINTAINED = 'N'
		) AND OWNER != 'OPS$ORACLE'
		ORDER BY OWNER, TABLE_NAME, INDEX_NAME`
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
func getViews(txn *sql.Tx) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := `
		SELECT OWNER, VIEW_NAME, TEXT
		FROM sys.all_views
		WHERE OWNER IN (
			SELECT USERNAME FROM DBA_USERS WHERE ORACLE_MAINTAINED = 'N'
		) AND OWNER != 'OPS$ORACLE'
		ORDER BY owner, view_name
	`
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
