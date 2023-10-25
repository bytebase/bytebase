package obo

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const systemSchemas = "'DWEXP','OMC','ORAAUDITOR','LBACSYS','SYS'"

var semVersionRegex = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`)

func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var version string
	if err := driver.db.QueryRowContext(ctx, "SELECT OB_VERSION() FROM DUAL").Scan(&version); err != nil {
		return nil, errors.Wrapf(err, "failed to get version")
	}

	query := fmt.Sprintf(`
		SELECT username FROM sys.all_users
		WHERE username NOT IN (%s)
		ORDER BY username
	`, systemSchemas)
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	instance := &db.InstanceMetadata{
		Version:   version,
		Databases: nil,
	}
	for _, schema := range schemas {
		instance.Databases = append(instance.Databases, &storepb.DatabaseSchemaMetadata{
			Name: schema,
		})
	}
	return instance, nil
}

func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	tx, err := driver.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	tableMap, err := getTables(ctx, tx, driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
	}
	viewMap, err := getViews(ctx, tx, driver.databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name: driver.databaseName,
	}
	databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
		Name:   "",
		Tables: tableMap[driver.databaseName],
		Views:  viewMap[driver.databaseName],
	})
	return databaseMetadata, nil
}

func getTables(ctx context.Context, tx *sql.Tx, schemaName string) (map[string][]*storepb.TableMetadata, error) {
	columnMap, err := getTableColumns(ctx, tx, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table columns")
	}
	indexMap, err := getIndexes(ctx, tx, schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indexes")
	}
	// TODO(d): foreign keys.
	tableMap := make(map[string][]*storepb.TableMetadata)
	query := fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, NUM_ROWS
		FROM all_tables
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME`, schemaName)

	rows, err := tx.QueryContext(ctx, query)
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

func getTableColumns(ctx context.Context, tx *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	query := fmt.Sprintf(`
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

	rows, err := tx.QueryContext(ctx, query)
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
			// TODO: use correct default type
			column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: defaultStr.String}
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

func getIndexes(ctx context.Context, tx *sql.Tx, schemaName string) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	expressionsMap := make(map[db.IndexKey][]string)

	queryColumn := fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_NAME
		FROM sys.all_ind_columns
		WHERE TABLE_OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, schemaName)

	colRows, err := tx.QueryContext(ctx, queryColumn)
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
	queryExpression := fmt.Sprintf(`
		SELECT TABLE_OWNER, TABLE_NAME, INDEX_NAME, COLUMN_EXPRESSION, COLUMN_POSITION
		FROM sys.all_ind_expressions
		WHERE TABLE_OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME, COLUMN_POSITION`, schemaName)
	expRows, err := tx.QueryContext(ctx, queryExpression)
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
	query := fmt.Sprintf(`
		SELECT OWNER, TABLE_NAME, INDEX_NAME, UNIQUENESS, INDEX_TYPE
		FROM sys.all_indexes
		WHERE OWNER = '%s'
		ORDER BY TABLE_NAME, INDEX_NAME`, schemaName)
	rows, err := tx.QueryContext(ctx, query)
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

		key := db.TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexMap, nil
}

func getViews(ctx context.Context, tx *sql.Tx, schemaName string) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := fmt.Sprintf(`
		SELECT OWNER, VIEW_NAME, TEXT
		FROM sys.all_views
		WHERE OWNER = '%s'
		ORDER BY view_name
	`, schemaName)

	rows, err := tx.QueryContext(ctx, query)
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

func (*Driver) SyncSlowQuery(context.Context, time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) CheckSlowQueryLogEnabled(context.Context) error {
	return errors.New("not implemented")
}
