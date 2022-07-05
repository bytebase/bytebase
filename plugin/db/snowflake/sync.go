package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
)

var (
	systemSchemas = map[string]bool{
		"information_schema": true,
	}
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMeta, error) {
	// Query user info
	if err := driver.useRole(ctx, accountAdminRole); err != nil {
		return nil, err
	}

	userList, err := driver.getUserList(ctx)
	if err != nil {
		return nil, err
	}

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, err
	}

	var databaseList []db.DatabaseMeta
	for _, database := range databases {
		if database == bytebaseDatabase {
			continue
		}

		databaseList = append(
			databaseList,
			db.DatabaseMeta{
				Name: database,
			},
		)
	}

	return &db.InstanceMeta{
		UserList:     userList,
		DatabaseList: databaseList,
	}, nil
}

// SyncSchema synces the schema.
func (driver *Driver) SyncSchema(ctx context.Context, databaseList ...string) ([]*db.Schema, error) {
	// Query user info
	if err := driver.useRole(ctx, accountAdminRole); err != nil {
		return nil, err
	}

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, err
	}

	var schemaList []*db.Schema
	for _, database := range databases {
		if database == bytebaseDatabase {
			continue
		}
		if len(databaseList) != 0 {
			exists := false
			for _, k := range databaseList {
				if database == k {
					exists = true
					break
				}
			}
			if !exists {
				continue
			}
		}

		var schema db.Schema
		schema.Name = database
		tableList, viewList, err := driver.syncTableSchema(ctx, database)
		if err != nil {
			return nil, err
		}
		schema.TableList, schema.ViewList = tableList, viewList

		schemaList = append(schemaList, &schema)
	}

	return schemaList, nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]db.User, error) {
	query := `
		SELECT
			GRANTEE_NAME,
			ROLE
		FROM SNOWFLAKE.ACCOUNT_USAGE.GRANTS_TO_USERS
`
	grants := make(map[string][]string)

	grantRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer grantRows.Close()

	for grantRows.Next() {
		var name, role string
		if err := grantRows.Scan(
			&name,
			&role,
		); err != nil {
			return nil, err
		}
		grants[name] = append(grants[name], role)
	}

	// Query user info
	query = `
	  SELECT
			name
		FROM SNOWFLAKE.ACCOUNT_USAGE.USERS
	`
	var userList []db.User
	userRows, err := driver.db.QueryContext(ctx, query)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer userRows.Close()

	for userRows.Next() {
		var name string
		if err := userRows.Scan(
			&name,
		); err != nil {
			return nil, err
		}

		userList = append(userList, db.User{
			Name:  name,
			Grant: strings.Join(grants[name], ", "),
		})
	}
	return userList, nil
}

func (driver *Driver) syncTableSchema(ctx context.Context, database string) ([]db.Table, []db.View, error) {
	// Query table info
	var excludedSchemaList []string

	// Skip all system schemas.
	for k := range systemSchemas {
		excludedSchemaList = append(excludedSchemaList, fmt.Sprintf("'%s'", k))
	}
	excludeWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedSchemaList, ", "))

	// Query column info
	query := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			COLUMN_DEFAULT,
			IS_NULLABLE,
			DATA_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			IFNULL(COMMENT, '')
		FROM %s.INFORMATION_SCHEMA.COLUMNS
		WHERE %s`, database, excludeWhere)
	columnRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer columnRows.Close()

	// schemaName.tableName -> columnList map
	columnMap := make(map[string][]db.Column)
	for columnRows.Next() {
		var schemaName string
		var tableName string
		var nullable string
		var defaultStr sql.NullString
		var column db.Column
		if err := columnRows.Scan(
			&schemaName,
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

		key := fmt.Sprintf("%s.%s", schemaName, tableName)
		columnMap[key] = append(columnMap[key], column)
	}

	query = fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			DATE_PART(EPOCH_SECOND, CREATED),
			DATE_PART(EPOCH_SECOND, LAST_ALTERED),
			TABLE_TYPE,
			ROW_COUNT,
			BYTES,
			IFNULL(COMMENT, '')
		FROM %s.INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE' AND %s`, database, excludeWhere)
	tableRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer tableRows.Close()

	var tables []db.Table
	for tableRows.Next() {
		var schemaName, tableName string
		var table db.Table
		if err := tableRows.Scan(
			&schemaName,
			&tableName,
			&table.CreatedTs,
			&table.UpdatedTs,
			&table.Type,
			&table.RowCount,
			&table.DataSize,
			&table.Comment,
		); err != nil {
			return nil, nil, err
		}

		table.Name = fmt.Sprintf("%s.%s", schemaName, tableName)
		table.ColumnList = columnMap[table.Name]
		tables = append(tables, table)
	}
	if err := tableRows.Err(); err != nil {
		return nil, nil, err
	}

	query = fmt.Sprintf(`
	SELECT
		TABLE_SCHEMA,
		TABLE_NAME,
		DATE_PART(EPOCH_SECOND, CREATED),
		DATE_PART(EPOCH_SECOND, LAST_ALTERED),
		IFNULL(VIEW_DEFINITION, ''),
		IFNULL(COMMENT, '')
	FROM %s.INFORMATION_SCHEMA.VIEWS
	WHERE %s`, database, excludeWhere)
	viewRows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, query)
	}
	defer viewRows.Close()

	var views []db.View
	for viewRows.Next() {
		var schemaName, viewName string
		var createdTs, updatedTs sql.NullInt64
		var view db.View
		if err := viewRows.Scan(
			&schemaName,
			&viewName,
			&createdTs,
			&updatedTs,
			&view.Definition,
			&view.Comment,
		); err != nil {
			return nil, nil, err
		}
		view.Name = fmt.Sprintf("%s.%s", schemaName, viewName)
		if createdTs.Valid {
			view.CreatedTs = createdTs.Int64
		}
		if updatedTs.Valid {
			view.UpdatedTs = updatedTs.Int64
		}
		views = append(views, view)
	}
	if err := viewRows.Err(); err != nil {
		return nil, nil, err
	}

	return tables, views, nil
}
