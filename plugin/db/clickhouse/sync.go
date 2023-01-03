package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"

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
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	// Query column info
	// tableName -> columnList map
	columnMap := make(map[string][]*storepb.ColumnMetadata)
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
	for columnRows.Next() {
		var tableName string
		var defaultStr sql.NullString
		column := &storepb.ColumnMetadata{}
		if err := columnRows.Scan(
			&tableName,
			&column.Name,
			&column.Position,
			&defaultStr,
			&column.Type,
			&column.Comment,
		); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
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
				Definition: definition,
				Comment:    comment,
			})
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	return &storepb.DatabaseMetadata{
		Name:    databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
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
