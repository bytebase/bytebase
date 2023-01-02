package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"

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

	databaseMetadata := &storepb.DatabaseMetadata{
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

	tableMap, viewMap, err := driver.getTableSchema(ctx, databaseName)
	if err != nil {
		return nil, err
	}
	schemaNameMap := make(map[string]bool)
	for schemaName := range tableMap {
		schemaNameMap[schemaName] = true
	}
	for schemaName := range viewMap {
		schemaNameMap[schemaName] = true
	}
	var schemaNames []string
	for schemaName := range schemaNameMap {
		schemaNames = append(schemaNames, schemaName)
	}
	sort.Strings(schemaNames)
	for _, schemaName := range schemaNames {
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tableMap[schemaName],
			Views:  viewMap[schemaName],
		})
	}

	return databaseMetadata, nil
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

func (driver *Driver) getTableSchema(ctx context.Context, database string) (map[string][]*storepb.TableMetadata, map[string][]*storepb.ViewMetadata, error) {
	tableMap, viewMap := make(map[string][]*storepb.TableMetadata), make(map[string][]*storepb.ViewMetadata)

	// Query table info
	var excludedSchemaList []string
	// Skip all system schemas.
	for k := range systemSchemas {
		excludedSchemaList = append(excludedSchemaList, fmt.Sprintf("'%s'", k))
	}
	excludeWhere := fmt.Sprintf("LOWER(TABLE_SCHEMA) NOT IN (%s)", strings.Join(excludedSchemaList, ", "))

	// Query column info.
	columnMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
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
		WHERE %s
		ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION`, database, excludeWhere)
	columnRows, err := driver.db.QueryContext(ctx, columnQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()
	for columnRows.Next() {
		var schemaName, tableName, nullable string
		var defaultStr sql.NullString
		column := &storepb.ColumnMetadata{}
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
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, nil, err
		}
		column.Nullable = isNullBool

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnMap[key] = append(columnMap[key], column)
	}
	if err := columnRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, columnQuery)
	}

	tableQuery := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			ROW_COUNT,
			BYTES,
			IFNULL(COMMENT, '')
		FROM %s.INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE' AND %s
		ORDER BY TABLE_SCHEMA, TABLE_NAME`, database, excludeWhere)
	tableRows, err := driver.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var schemaName string
		table := &storepb.TableMetadata{}
		if err := tableRows.Scan(
			&schemaName,
			&table.Name,
			&table.RowCount,
			&table.DataSize,
			&table.Comment,
		); err != nil {
			return nil, nil, err
		}

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := tableRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, tableQuery)
	}

	viewQuery := fmt.Sprintf(`
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			IFNULL(VIEW_DEFINITION, ''),
			IFNULL(COMMENT, '')
		FROM %s.INFORMATION_SCHEMA.VIEWS
		WHERE %s
		ORDER BY TABLE_SCHEMA, TABLE_NAME`, database, excludeWhere)
	viewRows, err := driver.db.QueryContext(ctx, viewQuery)
	if err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
	}
	defer viewRows.Close()
	for viewRows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		if err := viewRows.Scan(
			&schemaName,
			&view.Name,
			&view.Definition,
			&view.Comment,
		); err != nil {
			return nil, nil, err
		}

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := viewRows.Err(); err != nil {
		return nil, nil, util.FormatErrorWithQuery(err, viewQuery)
	}

	return tableMap, viewMap, nil
}
