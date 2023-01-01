package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

	var databaseList []db.DatabaseMeta
	// Query db info
	where := fmt.Sprintf("name NOT IN (%s)", strings.Join(excludedDatabaseList, ", "))
	query := `
		SELECT
			name
		FROM system.databases
		WHERE ` + where
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()
	for rows.Next() {
		var databaseMeta db.DatabaseMeta
		if err := rows.Scan(
			&databaseMeta.Name,
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
func (driver *Driver) SyncDBSchema(ctx context.Context, databaseName string) (*storepb.DatabaseMetadata, error) {
	// Query column info
	columnQuery := `
		SELECT
			table,
			name,
			position,
			default_expression,
			type,
			comment
		FROM system.columns
		WHERE database = $1
		ORDER BY table, position`
	columnRows, err := driver.db.QueryContext(ctx, columnQuery, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()

	// tableName -> columnList map
	columnMap := make(map[string][]db.Column)
	for columnRows.Next() {
		var tableName string
		var column db.Column
		if err := columnRows.Scan(
			&tableName,
			&column.Name,
			&column.Position,
			&column.Default,
			&column.Type,
			&column.Comment,
		); err != nil {
			return nil, err
		}

		columnMap[tableName] = append(columnMap[tableName], column)
	}
	if err := columnRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	// Query table info
	tableQuery := `
			SELECT
				name,
				engine,
				IFNULL(total_rows, 0),
				IFNULL(total_bytes, 0),
				metadata_modification_time,
				create_table_query,
				comment
			FROM system.tables
			WHERE database = $1
			ORDER BY name`
	tableRows, err := driver.db.QueryContext(ctx, tableQuery, databaseName)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()

	var tableList []db.Table
	var viewList []*storepb.ViewMetadata

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

		if engine == "View" {
			viewList = append(viewList, &storepb.ViewMetadata{
				Name:       name,
				Definition: definition,
				Comment:    comment,
			})
		} else {
			var table db.Table
			table.Type = "BASE TABLE"
			table.Name = name
			table.ShortName = name
			table.Engine = engine
			table.Comment = comment
			table.RowCount = rowCount
			table.DataSize = totalBytes
			table.UpdatedTs = lastUpdatedTime.Unix()
			table.ColumnList = columnMap[name]
			tableList = append(tableList, table)
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	var schema db.Schema
	schema.Name = databaseName
	schema.TableList = tableList
	databaseMetadata := util.ConvertDBSchema(&schema)

	if len(viewList) > 0 {
		if len(databaseMetadata.Schemas) == 0 {
			databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
				Name: "",
			})
		}
		databaseMetadata.Schemas[0].Views = viewList
	}

	return databaseMetadata, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]db.User, error) {
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

	var userList []db.User
	for userRows.Next() {
		var user string
		if err := userRows.Scan(
			&user,
		); err != nil {
			return nil, err
		}

		if err := func() error {
			// Uses single quote instead of backtick to escape because this is a string
			// instead of table (which should use backtick instead). MySQL actually works
			// in both ways. On the other hand, some other MySQL compatible engines might not (OceanBase in this case).
			grantQuery := fmt.Sprintf("SHOW GRANTS FOR %s", user)
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
				Name:  user,
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
