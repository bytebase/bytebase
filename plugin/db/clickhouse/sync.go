package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
)

// SyncSchema syncs the schema.
func (driver *Driver) SyncSchema(ctx context.Context) ([]*db.User, []*db.Schema, error) {
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

	// Query column info
	columnWhere := fmt.Sprintf("LOWER(database) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	columnQuery := `
			SELECT
				database,
				table,
				name,
				position,
				default_expression,
				type,
				comment
			FROM system.columns
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
		var column db.Column
		if err := columnRows.Scan(
			&dbName,
			&tableName,
			&column.Name,
			&column.Position,
			&column.Default,
			&column.Type,
			&column.Comment,
		); err != nil {
			return nil, nil, err
		}

		key := fmt.Sprintf("%s/%s", dbName, tableName)
		if tableList, ok := columnMap[key]; ok {
			columnMap[key] = append(tableList, column)
		} else {
			columnMap[key] = append([]db.Column(nil), column)
		}
	}
	if err := columnRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	// Query table info
	tableWhere := fmt.Sprintf("LOWER(database) NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	tableQuery := `
			SELECT
				database,
				name,
				engine,
				IFNULL(total_rows, 0),
				IFNULL(total_bytes, 0),
				metadata_modification_time,
				create_table_query,
				comment
			FROM system.tables
			WHERE ` + tableWhere
	tableRows, err := driver.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()

	// dbName -> tableList map
	tableMap := make(map[string][]db.Table)
	// dbName -> viewList map
	viewMap := make(map[string][]db.View)

	for tableRows.Next() {
		var dbName, name, engine, definition, comment string
		var rowCount, totalBytes int64
		var lastUpdatedTime time.Time
		if err := tableRows.Scan(
			&dbName,
			&name,
			&engine,
			&rowCount,
			&totalBytes,
			&lastUpdatedTime,
			&definition,
			&comment,
		); err != nil {
			return nil, nil, err
		}

		if engine == "View" {
			var view db.View
			view.Name = name
			view.UpdatedTs = lastUpdatedTime.Unix()
			view.Definition = definition
			view.Comment = comment
			viewMap[dbName] = append(viewMap[dbName], view)
		} else {
			var table db.Table
			table.Type = "BASE TABLE"
			table.Name = name
			table.Engine = engine
			table.Comment = comment
			table.RowCount = rowCount
			table.DataSize = totalBytes
			table.UpdatedTs = lastUpdatedTime.Unix()
			key := fmt.Sprintf("%s/%s", dbName, name)
			table.ColumnList = columnMap[key]
			tableMap[dbName] = append(tableMap[dbName], table)
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	var schemaList []*db.Schema
	// Query db info
	databaseWhere := fmt.Sprintf("name NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	databaseQuery := `
		SELECT
			name
		FROM system.databases
		WHERE ` + databaseWhere
	databaseRows, err := driver.db.QueryContext(ctx, databaseQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, databaseQuery)
	}
	defer databaseRows.Close()
	for databaseRows.Next() {
		var schema db.Schema
		if err := databaseRows.Scan(
			&schema.Name,
		); err != nil {
			return nil, nil, err
		}
		schema.TableList = tableMap[schema.Name]
		schema.ViewList = viewMap[schema.Name]

		schemaList = append(schemaList, &schema)
	}
	if err := databaseRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, databaseQuery)
	}

	return userList, schemaList, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]*db.User, error) {
	// Query user info
	// host_ip isn't used for user identifier.
	userQuery := `
	  SELECT
			name
		FROM system.users
	`
	userRows, err := driver.db.QueryContext(ctx, userQuery)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
	}
	defer userRows.Close()

	var userList []*db.User
	for userRows.Next() {
		var user string
		if err := userRows.Scan(
			&user,
		); err != nil {
			return nil, err
		}

		// Uses single quote instead of backtick to escape because this is a string
		// instead of table (which should use backtick instead). MySQL actually works
		// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
		grantQuery := fmt.Sprintf("SHOW GRANTS FOR %s", user)
		grantRows, err := driver.db.QueryContext(ctx,
			grantQuery,
		)
		if err != nil {
			return nil, util.FormatErrorWithQuery(err, grantQuery)
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
		if err := grantRows.Err(); err != nil {
			return nil, util.FormatErrorWithQuery(err, grantQuery)
		}

		userList = append(userList, &db.User{
			Name:  user,
			Grant: strings.Join(grantList, "\n"),
		})
	}
	if err := userRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
	}
	return userList, nil
}
