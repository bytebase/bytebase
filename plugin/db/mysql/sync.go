package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
)

// SyncSchema syncs the schema.
func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.User, []*db.Schema, error) {
	// Query MySQL version
	version, err := driver.GetVersion(ctx)
	if err != nil {
		return nil, nil, err
	}
	isMySQL8 := strings.HasPrefix(version, "8.0")

	excludedDatabaseList := []string{
		// Skip our internal "bytebase" database
		"'bytebase'",
	}

	// Skip all system databases
	for k := range systemDatabases {
		excludedDatabaseList = append(excludedDatabaseList, fmt.Sprintf("'%s'", k))
	}

	// Query user info
	userList, err := driver.getUserList(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Query index info
	indexWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query := `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				'',
				SEQ_IN_INDEX,
				INDEX_TYPE,
				CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
				1,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE ` + indexWhere
	if isMySQL8 {
		query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				EXPRESSION,
				SEQ_IN_INDEX,
				INDEX_TYPE,
				CASE NON_UNIQUE WHEN 0 THEN 1 ELSE 0 END AS IS_UNIQUE,
				CASE IS_VISIBLE WHEN 'YES' THEN 1 ELSE 0 END,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE ` + indexWhere
	}
	indexRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer indexRows.Close()

	// dbName/tableName -> indexList map
	indexMap := make(map[string][]db.Index)
	for indexRows.Next() {
		var dbName string
		var tableName string
		var columnName sql.NullString
		var expression sql.NullString
		var index db.Index
		if err := indexRows.Scan(
			&dbName,
			&tableName,
			&index.Name,
			&columnName,
			&expression,
			&index.Position,
			&index.Type,
			&index.Unique,
			&index.Visible,
			&index.Comment,
		); err != nil {
			return nil, nil, err
		}

		if columnName.Valid {
			index.Expression = columnName.String
		} else if expression.Valid {
			index.Expression = expression.String
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if indexList, ok := indexMap[key]; ok {
			indexMap[key] = append(indexList, index)
		} else {
			indexMap[key] = []db.Index{index}
		}
	}

	// Query column info
	columnWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				IFNULL(COLUMN_NAME, ''),
				ORDINAL_POSITION,
				COLUMN_DEFAULT,
				IS_NULLABLE,
				COLUMN_TYPE,
				IFNULL(CHARACTER_SET_NAME, ''),
				IFNULL(COLLATION_NAME, ''),
				COLUMN_COMMENT
			FROM information_schema.COLUMNS
			WHERE ` + columnWhere
	columnRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer columnRows.Close()

	// dbName/tableName -> columnList map
	columnMap := make(map[string][]db.Column)
	for columnRows.Next() {
		var dbName string
		var tableName string
		var nullable string
		var defaultStr sql.NullString
		var column db.Column
		if err := columnRows.Scan(
			&dbName,
			&tableName,
			&column.Name,
			&column.Position,
			&defaultStr,
			&nullable,
			&column.Type,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
		); err != nil {
			return nil, nil, err
		}

		if defaultStr.Valid {
			column.Default = &defaultStr.String
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if tableList, ok := columnMap[key]; ok {
			columnMap[key] = append(tableList, column)
		} else {
			columnMap[key] = []db.Column{column}
		}
	}

	// Query table info
	tableWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				IFNULL(UNIX_TIMESTAMP(CREATE_TIME), 0),
				IFNULL(UNIX_TIMESTAMP(UPDATE_TIME), 0),
				TABLE_TYPE,
				IFNULL(ENGINE, ''),
				IFNULL(TABLE_COLLATION, ''),
				IFNULL(TABLE_ROWS, 0),
				IFNULL(DATA_LENGTH, 0),
				IFNULL(INDEX_LENGTH, 0),
				IFNULL(DATA_FREE, 0),
				IFNULL(CREATE_OPTIONS, ''),
				IFNULL(TABLE_COMMENT, '')
			FROM information_schema.TABLES
			WHERE ` + tableWhere
	tableRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer tableRows.Close()

	// dbName -> tableList map
	tableMap := make(map[string][]db.Table)
	type ViewInfo struct {
		createdTs int64
		updatedTs int64
		comment   string
	}
	// dbName/viewName -> ViewInfo
	viewInfoMap := make(map[string]ViewInfo)
	for tableRows.Next() {
		var dbName string
		// Workaround TiDB bug https://github.com/pingcap/tidb/issues/27970
		var tableCollation sql.NullString
		var table db.Table
		if err := tableRows.Scan(
			&dbName,
			&table.Name,
			&table.CreatedTs,
			&table.UpdatedTs,
			&table.Type,
			&table.Engine,
			&tableCollation,
			&table.RowCount,
			&table.DataSize,
			&table.IndexSize,
			&table.DataFree,
			&table.CreateOptions,
			&table.Comment,
		); err != nil {
			return nil, nil, err
		}

		switch table.Type {
		case baseTableType:
			if tableCollation.Valid {
				table.Collation = tableCollation.String
			}

			key := fmt.Sprintf("%s/%s", dbName, table.Name)
			table.ColumnList = columnMap[key]
			table.IndexList = indexMap[key]

			if tableList, ok := tableMap[dbName]; ok {
				tableMap[dbName] = append(tableList, table)
			} else {
				tableMap[dbName] = []db.Table{table}
			}
		case viewTableType:
			viewInfoMap[fmt.Sprintf("%s/%s", dbName, table.Name)] = ViewInfo{
				createdTs: table.CreatedTs,
				updatedTs: table.UpdatedTs,
				comment:   table.Comment,
			}
		}
	}

	// Query view info
	viewWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				VIEW_DEFINITION
			FROM information_schema.VIEWS
			WHERE ` + viewWhere
	viewRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer viewRows.Close()

	// dbName -> viewList map
	viewMap := make(map[string][]db.View)
	for viewRows.Next() {
		var dbName string
		var view db.View
		if err := viewRows.Scan(
			&dbName,
			&view.Name,
			&view.Definition,
		); err != nil {
			return nil, nil, err
		}

		info := viewInfoMap[fmt.Sprintf("%s/%s", dbName, view.Name)]
		view.CreatedTs = info.createdTs
		view.UpdatedTs = info.updatedTs
		view.Comment = info.comment

		if viewList, ok := viewMap[dbName]; ok {
			viewMap[dbName] = append(viewList, view)
		} else {
			viewMap[dbName] = []db.View{view}
		}
	}

	// Query db info
	where := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query = `
			SELECT
		    SCHEMA_NAME,
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE ` + where
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var schemaList []*db.Schema
	for rows.Next() {
		var schema db.Schema
		if err := rows.Scan(
			&schema.Name,
			&schema.CharacterSet,
			&schema.Collation,
		); err != nil {
			return nil, nil, err
		}

		schema.TableList = tableMap[schema.Name]
		schema.ViewList = viewMap[schema.Name]

		schemaList = append(schemaList, &schema)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return userList, schemaList, err
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.User, error) {
	// Query user info
	query := `
	  SELECT
			user,
			host
		FROM mysql.user
		WHERE user NOT LIKE 'mysql.%'
	`
	var userList []*db.User
	userRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer userRows.Close()

	for userRows.Next() {
		var user string
		var host string
		if err := userRows.Scan(
			&user,
			&host,
		); err != nil {
			return nil, err
		}

		// Uses single quote instead of backtick to escape because this is a string
		// instead of table (which should use backtick instead). MySQL actually works
		// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
		name := fmt.Sprintf("'%s'@'%s'", user, host)
		query = fmt.Sprintf("SHOW GRANTS FOR %s", name)
		grantRows, err := driver.db.QueryContext(ctx,
			query,
		)
		if err != nil {
			return nil, util.FormatErrorWithQuery(err, query)
		}
		defer grantRows.Close()

		grantList := []string{}
		for grantRows.Next() {
			var grant string
			if err := grantRows.Scan(&grant); err != nil {
				return nil, err
			}
			grantList = append(grantList, grant)
		}

		userList = append(userList, &db.User{
			Name:  name,
			Grant: strings.Join(grantList, "\n"),
		})
	}
	return userList, nil
}
