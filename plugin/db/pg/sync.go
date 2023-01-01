package pg

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// pgDatabaseSchema describes a pg database schema.
type pgDatabaseSchema struct {
	name     string
	encoding string
	collate  string
}

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

	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}
	var databaseList []db.DatabaseMeta
	for _, database := range databases {
		dbName := database.name
		// Skip all system databases
		if _, ok := excludedDatabaseList[dbName]; ok {
			continue
		}

		databaseList = append(
			databaseList,
			db.DatabaseMeta{
				Name:         dbName,
				CharacterSet: database.encoding,
				Collation:    database.collate,
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
	// Query db info
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	databaseMetadata := &storepb.DatabaseMetadata{
		Name: databaseName,
	}
	found := false
	for _, database := range databases {
		if database.name == databaseName {
			found = true
			databaseMetadata.CharacterSet = database.encoding
			databaseMetadata.Collation = database.collate
			break
		}
	}
	if !found {
		return nil, common.Errorf(common.NotFound, "database %q not found", databaseName)
	}

	sqldb, err := driver.GetDBConnection(ctx, databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database connection for %q", databaseName)
	}
	txn, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	tableMap, err := getTables(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", databaseName)
	}
	viewMap, err := getViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", databaseName)
	}
	extensions, err := getExtensions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get extensions from database %q", databaseName)
	}

	if err := txn.Commit(); err != nil {
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
	databaseMetadata.Extensions = extensions

	return databaseMetadata, err
}

func (driver *Driver) getUserList(ctx context.Context) ([]db.User, error) {
	// Query user info
	query := `
		SELECT r.rolname, r.rolsuper, r.rolinherit, r.rolcreaterole, r.rolcreatedb, r.rolcanlogin, r.rolreplication, r.rolvaliduntil, r.rolbypassrls
		FROM pg_catalog.pg_roles r
		WHERE r.rolname !~ '^pg_';
	`
	var userList []db.User
	rows, err := driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		var super, inherit, createRole, createDB, canLogin, replication, bypassRLS bool
		var rolValidUntil sql.NullString
		if err := rows.Scan(
			&role,
			&super,
			&inherit,
			&createRole,
			&createDB,
			&canLogin,
			&replication,
			&rolValidUntil,
			&bypassRLS,
		); err != nil {
			return nil, err
		}

		var attributes []string
		if super {
			attributes = append(attributes, "Superuser")
		}
		if !inherit {
			attributes = append(attributes, "No inheritance")
		}
		if createRole {
			attributes = append(attributes, "Create role")
		}
		if createDB {
			attributes = append(attributes, "Create DB")
		}
		if !canLogin {
			attributes = append(attributes, "Cannot login")
		}
		if replication {
			attributes = append(attributes, "Replication")
		}
		if rolValidUntil.Valid {
			attributes = append(attributes, fmt.Sprintf("Password valid until %s", rolValidUntil.String))
		}
		if bypassRLS {
			attributes = append(attributes, "Bypass RLS+")
		}

		userList = append(userList, db.User{
			Name:  role,
			Grant: strings.Join(attributes, ", "),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return userList, nil
}

func getForeignKeys(txn *sql.Tx) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	query := `
	SELECT
		n.nspname AS fk_schema,
		conrelid::regclass AS fk_table,
		conname AS fk_name,
		(SELECT nspname FROM pg_namespace JOIN pg_class ON pg_namespace.oid = pg_class.relnamespace WHERE c.confrelid = pg_class.oid) AS fk_ref_schema,
		confrelid::regclass AS fk_ref_table,
		confdeltype AS delete_option,
		confupdtype AS update_option,
		confmatchtype AS match_option,
		pg_get_constraintdef(c.oid) AS fk_def
	FROM
		pg_constraint c
		JOIN pg_namespace n ON n.oid = c.connamespace
	WHERE
		n.nspname NOT IN('pg_catalog', 'information_schema')
		AND c.contype = 'f'
	ORDER BY fk_schema, fk_table, fk_name;
	`
	foreignKeysMap := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fkMetadata storepb.ForeignKeyMetadata
		var fkSchema, fkTable, fkDefinition string
		if err := rows.Scan(
			&fkSchema,
			&fkTable,
			&fkMetadata.Name,
			&fkMetadata.ReferencedSchema,
			&fkMetadata.ReferencedTable,
			&fkMetadata.OnDelete,
			&fkMetadata.OnUpdate,
			&fkMetadata.MatchType,
			&fkDefinition,
		); err != nil {
			return nil, err
		}

		fkTable = formatTableNameFromRegclass(fkTable)
		fkMetadata.ReferencedTable = formatTableNameFromRegclass(fkMetadata.ReferencedTable)
		fkMetadata.OnDelete = convertForeignKeyActionCode(fkMetadata.OnDelete)
		fkMetadata.OnUpdate = convertForeignKeyActionCode(fkMetadata.OnUpdate)
		fkMetadata.MatchType = convertForeignKeyMatchType(fkMetadata.MatchType)

		if fkMetadata.Columns, fkMetadata.ReferencedColumns, err = getForeignKeyColumnsAndReferencedColumns(fkDefinition); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: fkSchema, Table: fkTable}
		foreignKeysMap[key] = append(foreignKeysMap[key], &fkMetadata)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return foreignKeysMap, nil
}

func convertForeignKeyMatchType(in string) string {
	switch in {
	case "f":
		return "FULL"
	case "p":
		return "PARTIAL"
	case "s":
		return "SIMPLE"
	default:
		return in
	}
}

func convertForeignKeyActionCode(in string) string {
	switch in {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return in
	}
}

func getForeignKeyColumnsAndReferencedColumns(definition string) ([]string, []string, error) {
	columnsRegexp := regexp.MustCompile(`FOREIGN KEY \((.*)\) REFERENCES (.*)\((.*)\)`)
	matches := columnsRegexp.FindStringSubmatch(definition)
	if len(matches) != 4 {
		return nil, nil, errors.Errorf("invalid foreign key definition: %q", definition)
	}
	columnList, err := getColumnList(matches[1])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid foreign key definition: %q", definition)
	}
	referencedColumnList, err := getColumnList(matches[3])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid foreign key definition: %q", definition)
	}

	return columnList, referencedColumnList, nil
}

func getColumnList(definition string) ([]string, error) {
	list := strings.Split(definition, ",")
	if len(list) == 0 {
		return nil, errors.Errorf("invalid column list definition: %q", definition)
	}
	var result []string
	for _, name := range list {
		name = strings.TrimSpace(name)
		name = strings.Trim(name, `"`)
		result = append(result, name)
	}
	return result, nil
}

func formatTableNameFromRegclass(name string) string {
	if strings.Contains(name, ".") {
		name = name[1+strings.Index(name, "."):]
	}
	return strings.Trim(name, `"`)
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
	foreignKeysMap, err := getForeignKeys(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get foreign keys")
	}

	tableMap := make(map[string][]*storepb.TableMetadata)
	query := `
		SELECT tbl.schemaname, tbl.tablename,
			pg_table_size(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
			pg_indexes_size(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
			GREATEST(pc.reltuples::bigint, 0::BIGINT) AS estimate,
			obj_description(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass) AS comment
		FROM pg_catalog.pg_tables tbl
		LEFT JOIN pg_class as pc ON pc.oid = format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass
		WHERE tbl.schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY tbl.schemaname, tbl.tablename;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		// var tbl tableSchema
		var schemaName string
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &table.Name, &table.DataSize, &table.IndexSize, &table.RowCount, &comment); err != nil {
			return nil, err
		}
		if comment.Valid {
			table.Comment = comment.String
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.ForeignKeys = foreignKeysMap[key]

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
			cols.table_schema,
			cols.table_name,
			cols.column_name,
			cols.data_type,
			cols.ordinal_position,
			cols.column_default,
			cols.is_nullable,
			cols.collation_name,
			cols.udt_schema,
			cols.udt_name,
			pg_catalog.col_description(format('%s.%s', quote_ident(table_schema), quote_ident(table_name))::regclass, cols.ordinal_position::int) as column_comment
		FROM INFORMATION_SCHEMA.COLUMNS AS cols
		WHERE cols.table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY cols.table_schema, cols.table_name, cols.ordinal_position;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, isNullable string
		var defaultStr, collation, udtSchema, udtName, comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &column.Type, &column.Position, &defaultStr, &isNullable, &collation, &udtSchema, &udtName, &comment); err != nil {
			return nil, err
		}
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(isNullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		switch column.Type {
		case "USER-DEFINED":
			column.Type = fmt.Sprintf("%s.%s", udtSchema.String, udtName.String)
		case "ARRAY":
			column.Type = udtName.String
		}
		column.Collation = collation.String
		column.Comment = comment.String

		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := `
		SELECT schemaname, viewname, definition, obj_description(format('%s.%s', quote_ident(schemaname), quote_ident(viewname))::regclass) FROM pg_catalog.pg_views
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema');`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		var def, comment sql.NullString
		if err := rows.Scan(&schemaName, &view.Name, &def, &comment); err != nil {
			return nil, err
		}
		// Return error on NULL view definition.
		// https://github.com/bytebase/bytebase/issues/343
		if !def.Valid {
			return nil, errors.Errorf("schema %q view %q has empty definition; please check whether proper privileges have been granted to Bytebase", schemaName, view.Name)
		}
		view.Definition = def.String
		if comment.Valid {
			view.Comment = comment.String
		}

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return viewMap, nil
}

// getExtensions gets all extensions of a database.
func getExtensions(txn *sql.Tx) ([]*storepb.ExtensionMetadata, error) {
	var extensions []*storepb.ExtensionMetadata

	query := `
		SELECT e.extname, e.extversion, n.nspname, c.description
		FROM pg_catalog.pg_extension e
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = e.extnamespace
		LEFT JOIN pg_catalog.pg_description c ON c.objoid = e.oid AND c.classoid = 'pg_catalog.pg_extension'::pg_catalog.regclass
		WHERE n.nspname != 'pg_catalog'
		ORDER BY e.extname;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e := &storepb.ExtensionMetadata{}
		if err := rows.Scan(&e.Name, &e.Version, &e.Schema, &e.Description); err != nil {
			return nil, err
		}
		extensions = append(extensions, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return extensions, nil
}

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	indexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	query := `
		SELECT idx.schemaname, idx.tablename, idx.indexname, idx.indexdef, (SELECT 1
			FROM information_schema.table_constraints
			WHERE constraint_schema = idx.schemaname
			AND constraint_name = idx.indexname
			AND table_schema = idx.schemaname
			AND table_name = idx.tablename
			AND constraint_type = 'PRIMARY KEY') AS primary,
			obj_description(format('%s.%s', quote_ident(idx.schemaname), quote_ident(idx.indexname))::regclass) AS comment
		FROM pg_indexes AS idx WHERE idx.schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY idx.schemaname, idx.tablename, idx.indexname;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		index := &storepb.IndexMetadata{}
		var schemaName, tableName, statement string
		var primary sql.NullInt32
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &tableName, &index.Name, &statement, &primary, &comment); err != nil {
			return nil, err
		}
		index.Type = getIndexMethodType(statement)
		index.Unique = strings.Contains(statement, " UNIQUE INDEX ")
		index.Expressions, err = getIndexColumnExpressions(statement)
		if err != nil {
			return nil, err
		}
		if primary.Valid && primary.Int32 == 1 {
			index.Primary = true
		}
		if comment.Valid {
			index.Comment = comment.String
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexMap, nil
}

func getIndexMethodType(stmt string) string {
	re := regexp.MustCompile(`USING (\w+) `)
	matches := re.FindStringSubmatch(stmt)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

func getIndexColumnExpressions(stmt string) ([]string, error) {
	rc := regexp.MustCompile(`\((.*)\)`)
	rm := rc.FindStringSubmatch(stmt)
	if len(rm) == 0 {
		return nil, errors.Errorf("invalid index statement: %q", stmt)
	}
	columnStmt := rm[1]

	var cols []string
	re := regexp.MustCompile(`\(\(.*\)\)`)
	for {
		if len(columnStmt) == 0 {
			break
		}
		// Get a token
		token := ""
		// Expression has format of "((exp))".
		if strings.HasPrefix(columnStmt, "((") {
			token = re.FindString(columnStmt)
		} else {
			i := strings.Index(columnStmt, ",")
			if i < 0 {
				token = columnStmt
			} else {
				token = columnStmt[:i]
			}
		}
		// Strip token
		if len(token) == 0 {
			return nil, errors.Errorf("invalid index statement: %q", stmt)
		}
		columnStmt = columnStmt[len(token):]
		cols = append(cols, strings.TrimSpace(token))

		// Trim space and remove a comma to prepare for the next tokenization.
		columnStmt = strings.TrimSpace(columnStmt)
		if len(columnStmt) > 0 && columnStmt[0] == ',' {
			columnStmt = columnStmt[1:]
		}
		columnStmt = strings.TrimSpace(columnStmt)
	}

	return cols, nil
}
