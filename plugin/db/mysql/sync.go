package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		// TiDB only
		"metrics_schema":     true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMeta, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	// Query user info
	userList, err := driver.getUserList(ctx)
	if err != nil {
		return nil, err
	}

	excludedDatabaseList := []string{
		// Skip our internal "bytebase" database
		"'bytebase'",
	}
	// Skip all system databases
	for k := range systemDatabases {
		excludedDatabaseList = append(excludedDatabaseList, fmt.Sprintf("'%s'", k))
	}

	// Query db info
	where := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query := `
		SELECT
			SCHEMA_NAME,
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE ` + where
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databaseList []db.DatabaseMeta
	for rows.Next() {
		var databaseMeta db.DatabaseMeta
		if err := rows.Scan(
			&databaseMeta.Name,
			&databaseMeta.CharacterSet,
			&databaseMeta.Collation,
		); err != nil {
			return nil, err
		}
		databaseList = append(databaseList, databaseMeta)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMeta{
		Version:      version,
		UserList:     userList,
		DatabaseList: databaseList,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context, databaseName string) (*db.Schema, map[string][]*storepb.ForeignKeyMetadata, error) {
	// Query MySQL version
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, nil, err
	}
	isMySQL8 := strings.HasPrefix(version, "8.0")

	// Query index info
	indexWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) = '%s'", strings.ToLower(databaseName))
	indexQuery := `
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
		indexQuery = `
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
	indexRows, err := driver.db.QueryContext(ctx, indexQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, indexQuery)
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

		if index.Name == "PRIMARY" {
			index.Primary = true
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if indexList, ok := indexMap[key]; ok {
			indexMap[key] = append(indexList, index)
		} else {
			indexMap[key] = []db.Index{index}
		}
	}
	if err := indexRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, indexQuery)
	}

	// Query column info
	columnWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) = '%s'", strings.ToLower(databaseName))
	columnQuery := `
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
	columnRows, err := driver.db.QueryContext(ctx, columnQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
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
		// TODO(d): use convertBoolFromYesNo() if possible.
		if nullable == "YES" {
			column.Nullable = true
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if tableList, ok := columnMap[key]; ok {
			columnMap[key] = append(tableList, column)
		} else {
			columnMap[key] = []db.Column{column}
		}
	}
	if err := columnRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	// Query table info
	tableWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) = '%s'", strings.ToLower(databaseName))
	tableQuery := `
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
	tableRows, err := driver.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
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
		table.ShortName = table.Name

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
	if err := tableRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	// Query view info
	viewWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) = '%s'", strings.ToLower(databaseName))
	viewQuery := `
			SELECT
				TABLE_SCHEMA,
				TABLE_NAME,
				VIEW_DEFINITION
			FROM information_schema.VIEWS
			WHERE ` + viewWhere
	viewRows, err := driver.db.QueryContext(ctx, viewQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
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
		view.ShortName = view.Name

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
	if err := viewRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
	}

	// Query db info
	databaseWhere := fmt.Sprintf("LOWER(SCHEMA_NAME) = '%s'", strings.ToLower(databaseName))
	databaseQuery := `
		SELECT
			SCHEMA_NAME,
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE ` + databaseWhere
	var schema db.Schema
	if err := driver.db.QueryRowContext(ctx, databaseQuery).Scan(
		&schema.Name,
		&schema.CharacterSet,
		&schema.Collation); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, common.Errorf(common.NotFound, "database %q not found", databaseName)
		}
		return nil, nil, err
	}
	schema.TableList = tableMap[schema.Name]
	schema.ViewList = viewMap[schema.Name]

	fkMap, err := driver.getForeignKeyList(ctx, databaseName)
	if err != nil {
		return nil, nil, err
	}
	return &schema, fkMap, err
}

func (driver *Driver) getForeignKeyList(ctx context.Context, databaseName string) (map[string][]*storepb.ForeignKeyMetadata, error) {
	fkQuery := fmt.Sprintf(`
		SELECT
			fks.TABLE_NAME,
			fks.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			'',
			fks.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME,
			fks.DELETE_RULE,
			fks.UPDATE_RULE,
			fks.MATCH_OPTION
		FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS fks
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
			ON fks.CONSTRAINT_SCHEMA = kcu.TABLE_SCHEMA
				AND fks.TABLE_NAME = kcu.TABLE_NAME
				AND fks.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
		WHERE LOWER(fks.CONSTRAINT_SCHEMA) = '%s'
		ORDER BY fks.TABLE_NAME, fks.CONSTRAINT_NAME, kcu.ORDINAL_POSITION;
	`, strings.ToLower(databaseName))

	fkRows, err := driver.db.QueryContext(ctx, fkQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, fkQuery)
	}
	defer fkRows.Close()
	fkMap := make(map[string][]*storepb.ForeignKeyMetadata)
	var buildingFk *storepb.ForeignKeyMetadata
	var buildingTable string
	for fkRows.Next() {
		var tableName string
		var fk storepb.ForeignKeyMetadata
		var column, referencedColumn string
		if err := fkRows.Scan(
			&tableName,
			&fk.Name,
			&column,
			&fk.ReferencedSchema,
			&fk.ReferencedTable,
			&referencedColumn,
			&fk.OnDelete,
			&fk.OnUpdate,
			&fk.MatchType,
		); err != nil {
			return nil, err
		}

		fk.Columns = append(fk.Columns, column)
		fk.ReferencedColumns = append(fk.ReferencedColumns, referencedColumn)
		if buildingFk == nil {
			buildingTable = tableName
			buildingFk = &fk
		} else {
			if tableName == buildingTable && buildingFk.Name == fk.Name {
				buildingFk.Columns = append(buildingFk.Columns, fk.Columns[0])
				buildingFk.ReferencedColumns = append(buildingFk.ReferencedColumns, fk.ReferencedColumns[0])
			} else {
				if fkList, ok := fkMap[buildingTable]; ok {
					fkMap[buildingTable] = append(fkList, buildingFk)
					buildingTable = tableName
					buildingFk = &fk
				} else {
					fkMap[buildingTable] = []*storepb.ForeignKeyMetadata{buildingFk}
				}
			}
		}
	}

	if buildingFk != nil {
		if fkList, ok := fkMap[buildingTable]; ok {
			fkMap[buildingTable] = append(fkList, buildingFk)
		} else {
			fkMap[buildingTable] = []*storepb.ForeignKeyMetadata{buildingFk}
		}
	}

	return fkMap, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]db.User, error) {
	// Query user info
	userQuery := `
	  SELECT
			user,
			host
		FROM mysql.user
		WHERE user NOT LIKE 'mysql.%'
	`
	var userList []db.User
	userRows, err := driver.db.QueryContext(ctx, userQuery)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
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

		if err := func() error {
			// Uses single quote instead of backtick to escape because this is a string
			// instead of table (which should use backtick instead). MySQL actually works
			// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
			name := fmt.Sprintf("'%s'@'%s'", user, host)
			grantQuery := fmt.Sprintf("SHOW GRANTS FOR %s", name)
			grantRows, err := driver.db.QueryContext(ctx,
				grantQuery,
			)
			if err != nil {
				return util.FormatErrorWithQuery(err, grantQuery)
			}
			defer grantRows.Close()

			grantList := []string{}
			for grantRows.Next() {
				var grant string
				if err := grantRows.Scan(&grant); err != nil {
					return err
				}
				grantList = append(grantList, grant)
			}
			if err := grantRows.Err(); err != nil {
				return util.FormatErrorWithQuery(err, grantQuery)
			}

			userList = append(userList, db.User{
				Name:  name,
				Grant: strings.Join(grantList, "\n"),
			})
			return nil
		}(); err != nil {
			return nil, err
		}
	}
	if err := userRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
	}
	return userList, nil
}
