package pg

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

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

// tableSchema describes the schema of a pg table.
type tableSchema struct {
	schemaName    string
	name          string
	tableowner    string
	comment       string
	rowCount      int64
	tableSizeByte int64
	indexSizeByte int64

	columns     []*columnSchema
	constraints []*tableConstraint
}

// columnSchema describes the schema of a pg table column.
type columnSchema struct {
	columnName             string
	dataType               string
	ordinalPosition        int
	characterMaximumLength string
	columnDefault          string
	isNullable             bool
	collationName          string
	comment                string
}

// tableConstraint describes constraint schema of a pg table.
type tableConstraint struct {
	name       string
	schemaName string
	tableName  string
	constraint string
}

// viewSchema describes the schema of a pg view.
type viewSchema struct {
	schemaName string
	name       string
	definition string
	comment    string
}

// indexSchema describes the schema of a pg index.
type indexSchema struct {
	schemaName string
	name       string
	tableName  string
	statement  string
	unique     bool
	primary    bool
	// methodType such as btree.
	methodType        string
	columnExpressions []string
	comment           string
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

	schema := db.Schema{
		Name: databaseName,
	}
	found := false
	for _, database := range databases {
		if database.name == databaseName {
			found = true
			schema.CharacterSet = database.encoding
			schema.Collation = database.collate
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

	// Index statements.
	indicesMap := make(map[string][]*indexSchema)
	indices, err := getIndices(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices from database %q", databaseName)
	}
	for _, idx := range indices {
		key := fmt.Sprintf("%s.%s", idx.schemaName, idx.tableName)
		indicesMap[key] = append(indicesMap[key], idx)
	}

	// Table statements.
	tables, err := getPgTables(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", databaseName)
	}
	for _, tbl := range tables {
		var dbTable db.Table
		dbTable.Name = fmt.Sprintf("%s.%s", tbl.schemaName, tbl.name)
		dbTable.Schema = tbl.schemaName
		dbTable.ShortName = tbl.name
		dbTable.Type = "BASE TABLE"
		dbTable.Comment = tbl.comment
		dbTable.RowCount = tbl.rowCount
		dbTable.DataSize = tbl.tableSizeByte
		dbTable.IndexSize = tbl.indexSizeByte
		for _, col := range tbl.columns {
			var dbColumn db.Column
			dbColumn.Name = col.columnName
			dbColumn.Position = col.ordinalPosition
			dbColumn.Default = &col.columnDefault
			dbColumn.Type = col.dataType
			dbColumn.Nullable = col.isNullable
			dbColumn.Collation = col.collationName
			dbColumn.Comment = col.comment
			dbTable.ColumnList = append(dbTable.ColumnList, dbColumn)
		}
		indices := indicesMap[dbTable.Name]
		for _, idx := range indices {
			for i, colExp := range idx.columnExpressions {
				var dbIndex db.Index
				dbIndex.Name = idx.name
				dbIndex.Expression = colExp
				dbIndex.Position = i + 1
				dbIndex.Type = idx.methodType
				dbIndex.Unique = idx.unique
				dbIndex.Primary = idx.primary
				dbIndex.Comment = idx.comment
				dbTable.IndexList = append(dbTable.IndexList, dbIndex)
			}
		}

		schema.TableList = append(schema.TableList, dbTable)
	}
	// View statements.
	views, err := getViews(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", databaseName)
	}
	for _, view := range views {
		var dbView db.View
		dbView.Name = fmt.Sprintf("%s.%s", view.schemaName, view.name)
		dbView.Schema = view.schemaName
		dbView.ShortName = view.name
		// Postgres does not store
		dbView.CreatedTs = time.Now().Unix()
		dbView.Definition = view.definition
		dbView.Comment = view.comment

		schema.ViewList = append(schema.ViewList, dbView)
	}
	// Extensions.
	extensions, err := getExtensions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get extensions from database %q", databaseName)
	}

	// Foreign keys.
	foreignKeysMap, err := getForeignKeys(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get foreign keys from database %q", databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := util.ConvertDBSchema(&schema)
	for _, schemaMetadata := range databaseMetadata.Schemas {
		for _, tableMetadata := range schemaMetadata.Tables {
			tableMetadata.ForeignKeys = foreignKeysMap[db.TableKey{Schema: schemaMetadata.Name, Table: tableMetadata.Name}]
		}
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
		AND c.contype = 'f';
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
func getPgTables(txn *sql.Tx) ([]*tableSchema, error) {
	constraints, err := getTableConstraints(txn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get table constraints")
	}
	columns, err := getTableColumns(txn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get table columns")
	}

	var tables []*tableSchema
	query := `
	SELECT tbl.schemaname, tbl.tablename, tbl.tableowner,
		pg_table_size(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
		pg_indexes_size(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass),
		GREATEST(pc.reltuples::bigint, 0::BIGINT) AS estimate,
		obj_description(format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass) AS comment
	FROM pg_catalog.pg_tables tbl
	LEFT JOIN pg_class as pc ON pc.oid = format('%s.%s', quote_ident(tbl.schemaname), quote_ident(tbl.tablename))::regclass
	WHERE tbl.schemaname NOT IN ('pg_catalog', 'information_schema');`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tbl tableSchema
		var schemaname, tablename, tableowner string
		var tableSizeByte, indexSizeByte, rowCountEstimate int64
		var comment sql.NullString
		if err := rows.Scan(&schemaname, &tablename, &tableowner, &tableSizeByte, &indexSizeByte, &rowCountEstimate, &comment); err != nil {
			return nil, err
		}
		tbl.schemaName = schemaname
		tbl.name = tablename
		tbl.tableowner = tableowner
		tbl.tableSizeByte = tableSizeByte
		tbl.indexSizeByte = indexSizeByte
		tbl.rowCount = rowCountEstimate
		if comment.Valid {
			tbl.comment = comment.String
		}

		tables = append(tables, &tbl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, tbl := range tables {
		key := fmt.Sprintf("%s.%s", tbl.schemaName, tbl.name)
		tbl.constraints = constraints[key]
		tbl.columns = columns[key]
	}
	return tables, nil
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx) (map[string][]*columnSchema, error) {
	query := `
	SELECT
		cols.table_schema,
		cols.table_name,
		cols.column_name,
		cols.data_type,
		cols.ordinal_position,
		cols.character_maximum_length,
		cols.column_default,
		cols.is_nullable,
		cols.collation_name,
		cols.udt_schema,
		cols.udt_name,
		pg_catalog.col_description(format('%s.%s', quote_ident(table_schema), quote_ident(table_name))::regclass, cols.ordinal_position::int) as column_comment
	FROM INFORMATION_SCHEMA.COLUMNS AS cols
	WHERE cols.table_schema NOT IN ('pg_catalog', 'information_schema');`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnsMap := make(map[string][]*columnSchema)
	for rows.Next() {
		var tableSchema, tableName, columnName, dataType, isNullable string
		var characterMaximumLength, columnDefault, collationName, udtSchema, udtName, comment sql.NullString
		var ordinalPosition int
		if err := rows.Scan(&tableSchema, &tableName, &columnName, &dataType, &ordinalPosition, &characterMaximumLength, &columnDefault, &isNullable, &collationName, &udtSchema, &udtName, &comment); err != nil {
			return nil, err
		}
		isNullBool, err := util.ConvertYesNo(isNullable)
		if err != nil {
			return nil, err
		}
		c := columnSchema{
			columnName:             columnName,
			dataType:               dataType,
			ordinalPosition:        ordinalPosition,
			characterMaximumLength: characterMaximumLength.String,
			columnDefault:          columnDefault.String,
			isNullable:             isNullBool,
			collationName:          collationName.String,
			comment:                comment.String,
		}
		switch dataType {
		case "USER-DEFINED":
			c.dataType = fmt.Sprintf("%s.%s", udtSchema.String, udtName.String)
		case "ARRAY":
			c.dataType = udtName.String
		}
		key := fmt.Sprintf("%s.%s", tableSchema, tableName)
		columnsMap[key] = append(columnsMap[key], &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columnsMap, nil
}

// getTableConstraints gets all table constraints of a database.
func getTableConstraints(txn *sql.Tx) (map[string][]*tableConstraint, error) {
	query := "" +
		"SELECT n.nspname, conrelid::regclass, conname, pg_get_constraintdef(c.oid) " +
		"FROM pg_constraint c " +
		"JOIN pg_namespace n ON n.oid = c.connamespace " +
		"WHERE n.nspname NOT IN ('pg_catalog', 'information_schema');"
	ret := make(map[string][]*tableConstraint)
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var constraint tableConstraint
		if err := rows.Scan(&constraint.schemaName, &constraint.tableName, &constraint.name, &constraint.constraint); err != nil {
			return nil, err
		}
		if strings.Contains(constraint.tableName, ".") {
			constraint.tableName = constraint.tableName[1+strings.Index(constraint.tableName, "."):]
		}
		key := fmt.Sprintf("%s.%s", constraint.schemaName, constraint.tableName)
		ret[key] = append(ret[key], &constraint)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx) ([]*viewSchema, error) {
	query := `
	SELECT schemaname, viewname, definition, obj_description(format('%s.%s', quote_ident(schemaname), quote_ident(viewname))::regclass) FROM pg_catalog.pg_views
	WHERE schemaname NOT IN ('pg_catalog', 'information_schema');`
	var views []*viewSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var view viewSchema
		var def, comment sql.NullString
		if err := rows.Scan(&view.schemaName, &view.name, &def, &comment); err != nil {
			return nil, err
		}
		// Return error on NULL view definition.
		// https://github.com/bytebase/bytebase/issues/343
		if !def.Valid {
			return nil, errors.Errorf("schema %q view %q has empty definition; please check whether proper privileges have been granted to Bytebase", view.schemaName, view.name)
		}
		view.definition = def.String
		if comment.Valid {
			view.comment = comment.String
		}
		views = append(views, &view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return views, nil
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
		var e storepb.ExtensionMetadata
		if err := rows.Scan(&e.Name, &e.Version, &e.Schema, &e.Description); err != nil {
			return nil, err
		}
		extensions = append(extensions, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return extensions, nil
}

// getIndices gets all indices of a database.
func getIndices(txn *sql.Tx) ([]*indexSchema, error) {
	query := `
	SELECT idx.schemaname, idx.tablename, idx.indexname, idx.indexdef, (SELECT 1
		FROM information_schema.table_constraints
		WHERE constraint_schema = idx.schemaname
		AND constraint_name = idx.indexname
		AND table_schema = idx.schemaname
		AND table_name = idx.tablename
		AND constraint_type = 'PRIMARY KEY') AS primary,
		obj_description(format('%s.%s', quote_ident(idx.schemaname), quote_ident(idx.indexname))::regclass) AS comment
	FROM pg_indexes AS idx WHERE idx.schemaname NOT IN ('pg_catalog', 'information_schema');`

	var indices []*indexSchema
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idx indexSchema
		var primary sql.NullInt32
		var comment sql.NullString
		if err := rows.Scan(&idx.schemaName, &idx.tableName, &idx.name, &idx.statement, &primary, &comment); err != nil {
			return nil, err
		}
		idx.unique = strings.Contains(idx.statement, " UNIQUE INDEX ")
		idx.methodType = getIndexMethodType(idx.statement)
		idx.columnExpressions, err = getIndexColumnExpressions(idx.statement)
		if err != nil {
			return nil, err
		}
		if primary.Valid && primary.Int32 == 1 {
			idx.primary = true
		}
		if comment.Valid {
			idx.comment = comment.String
		}
		indices = append(indices, &idx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indices, nil
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
