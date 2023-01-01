package snowflake

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

	version, err := driver.getVersion(ctx)
	if err != nil {
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
		Version:      version,
		UserList:     userList,
		DatabaseList: databaseList,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context, databaseName string) (*storepb.DatabaseMetadata, error) {
	// Query user info
	if err := driver.useRole(ctx, accountAdminRole); err != nil {
		return nil, err
	}

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, err
	}

	schema := db.Schema{
		Name: databaseName,
	}
	found := false
	for _, database := range databases {
		if database == databaseName {
			found = true
			break
		}
	}
	if !found {
		return nil, common.Errorf(common.NotFound, "database %q not found", databaseName)
	}

	tableList, viewList, err := driver.syncTableSchema(ctx, databaseName)
	if err != nil {
		return nil, err
	}
	schema.TableList, schema.ViewList = tableList, viewList

	return util.ConvertDBSchema(&schema, nil), nil
}

func (driver *Driver) getUserList(ctx context.Context) ([]db.User, error) {
	grantQuery := `
		SELECT
			GRANTEE_NAME,
			ROLE
		FROM SNOWFLAKE.ACCOUNT_USAGE.GRANTS_TO_USERS
	`
	grants := make(map[string][]string)

	grantRows, err := driver.db.QueryContext(ctx, grantQuery)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, grantQuery)
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
	if err := grantRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, grantQuery)
	}

	// Query user info
	userQuery := `
	  SELECT
			name
		FROM SNOWFLAKE.ACCOUNT_USAGE.USERS
	`
	var userList []db.User
	userRows, err := driver.db.QueryContext(ctx, userQuery)

	if err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
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
	if err := userRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, userQuery)
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
	columnQuery := fmt.Sprintf(`
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
	columnRows, err := driver.db.QueryContext(ctx, columnQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
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
	if err := columnRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	tableQuery := fmt.Sprintf(`
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
	tableRows, err := driver.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
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
		table.Schema = schemaName
		table.ShortName = tableName
		table.ColumnList = columnMap[table.Name]
		tables = append(tables, table)
	}
	if err := tableRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	viewQuery := fmt.Sprintf(`
	SELECT
		TABLE_SCHEMA,
		TABLE_NAME,
		DATE_PART(EPOCH_SECOND, CREATED),
		DATE_PART(EPOCH_SECOND, LAST_ALTERED),
		IFNULL(VIEW_DEFINITION, ''),
		IFNULL(COMMENT, '')
	FROM %s.INFORMATION_SCHEMA.VIEWS
	WHERE %s`, database, excludeWhere)
	viewRows, err := driver.db.QueryContext(ctx, viewQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
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
		view.Schema = schemaName
		view.ShortName = viewName
		if createdTs.Valid {
			view.CreatedTs = createdTs.Int64
		}
		if updatedTs.Valid {
			view.UpdatedTs = updatedTs.Int64
		}
		views = append(views, view)
	}
	if err := viewRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
	}

	return tables, views, nil
}
