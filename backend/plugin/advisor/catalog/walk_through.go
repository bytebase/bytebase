package catalog

import (
	"fmt"
	"strings"

	mysqlparser "github.com/bytebase/mysql-parser"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/sessionctx/stmtctx"
	"github.com/pingcap/tidb/types"

	mysqlbbparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	tidbbbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// WalkThroughErrorType is the type of WalkThroughError.
type WalkThroughErrorType int

const (
	// PrimaryKeyName is the string for PK.
	PrimaryKeyName string = "PRIMARY"
	// FullTextName is the string for FULLTEXT.
	FullTextName string = "FULLTEXT"
	// SpatialName is the string for SPATIAL.
	SpatialName string = "SPATIAL"

	// ErrorTypeUnsupported is the error for unsupported cases.
	ErrorTypeUnsupported WalkThroughErrorType = 1
	// ErrorTypeInternal is the error for internal errors.
	ErrorTypeInternal WalkThroughErrorType = 2
	// ErrorTypeInvalidStatement is the error type for invalid statement errors.
	ErrorTypeInvalidStatement WalkThroughErrorType = 3

	// 101 parse error type.

	// ErrorTypeParseError is the error in parsing.
	ErrorTypeParseError WalkThroughErrorType = 101
	// ErrorTypeDeparseError is the error in deparsing.
	ErrorTypeDeparseError WalkThroughErrorType = 102

	// 201 ~ 299 database error type.

	// ErrorTypeAccessOtherDatabase is the error that try to access other database.
	ErrorTypeAccessOtherDatabase = 201
	// ErrorTypeDatabaseIsDeleted is the error that try to access the deleted database.
	ErrorTypeDatabaseIsDeleted = 202

	// 301 ~ 399 table error type.

	// ErrorTypeTableExists is the error that table exists.
	ErrorTypeTableExists = 301
	// ErrorTypeTableNotExists is the error that table not exists.
	ErrorTypeTableNotExists = 302
	// ErrorTypeUseCreateTableAs is the error that using CREATE TABLE AS statements.
	ErrorTypeUseCreateTableAs = 303
	// ErrorTypeTableIsReferencedByView is the error that table is referenced by view.
	ErrorTypeTableIsReferencedByView = 304
	// ErrorTypeViewNotExists is the error that view not exists.
	ErrorTypeViewNotExists = 305

	// 401 ~ 499 column error type.

	// ErrorTypeColumnExists is the error that column exists.
	ErrorTypeColumnExists = 401
	// ErrorTypeColumnNotExists is the error that column not exists.
	ErrorTypeColumnNotExists = 402
	// ErrorTypeDropAllColumns is the error that dropping all columns in a table.
	ErrorTypeDropAllColumns = 403
	// ErrorTypeAutoIncrementExists is the error that auto_increment exists.
	ErrorTypeAutoIncrementExists = 404
	// ErrorTypeOnUpdateColumnNotDatetimeOrTimestamp is the error that the ON UPDATE column is not datetime or timestamp.
	ErrorTypeOnUpdateColumnNotDatetimeOrTimestamp = 405
	// ErrorTypeSetNullDefaultForNotNullColumn is the error that setting NULL default value for the NOT NULL column.
	ErrorTypeSetNullDefaultForNotNullColumn = 406
	// ErrorTypeInvalidColumnTypeForDefaultValue is the error that invalid column type for default value.
	ErrorTypeInvalidColumnTypeForDefaultValue = 407
	// ErrorTypeColumnIsReferencedByView is the error that column is referenced by view.
	ErrorTypeColumnIsReferencedByView = 408

	// 501 ~ 599 index error type.

	// ErrorTypePrimaryKeyExists is the error that PK exists.
	ErrorTypePrimaryKeyExists = 501
	// ErrorTypeIndexExists is the error that index exists.
	ErrorTypeIndexExists = 502
	// ErrorTypeIndexEmptyKeys is the error that index has empty keys.
	ErrorTypeIndexEmptyKeys = 503
	// ErrorTypePrimaryKeyNotExists is the error that PK does not exist.
	ErrorTypePrimaryKeyNotExists = 504
	// ErrorTypeIndexNotExists is the error that index does not exist.
	ErrorTypeIndexNotExists = 505
	// ErrorTypeIncorrectIndexName is the incorrect index name error.
	ErrorTypeIncorrectIndexName = 506
	// ErrorTypeSpatialIndexKeyNullable is the error that keys in spatial index are nullable.
	ErrorTypeSpatialIndexKeyNullable = 507

	// 601 ~ 699 insert statement error type.

	// ErrorTypeInsertColumnCountNotMatchValueCount is the error that column count doesn't match value count.
	ErrorTypeInsertColumnCountNotMatchValueCount = 601
	// ErrorTypeInsertSpecifiedColumnTwice is the error that column specified twice in INSERT.
	ErrorTypeInsertSpecifiedColumnTwice = 602
	// ErrorTypeInsertNullIntoNotNullColumn is the error that insert NULL into NOT NULL columns.
	ErrorTypeInsertNullIntoNotNullColumn = 603

	// 701 ~ 799 schema error type.

	// ErrorTypeSchemaNotExists is the error that schema does not exist.
	ErrorTypeSchemaNotExists = 701
	// ErrorTypeSchemaExists is the error that schema already exists.
	ErrorTypeSchemaExists = 702

	// 801 ~ 899 relation error type.

	// ErrorTypeRelationExists is the error that relation already exists.
	ErrorTypeRelationExists = 801

	// 901 ~ 999 constraint error type.

	// ErrorTypeConstraintNotExists is the error that constraint doesn't exist.
	ErrorTypeConstraintNotExists = 901
)

// WalkThroughError is the error for walking-through.
type WalkThroughError struct {
	Type    WalkThroughErrorType
	Content string
	Line    int

	Payload any
}

// NewParseError returns a new ErrorTypeParseError.
func NewParseError(content string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeParseError,
		Content: content,
	}
}

// NewRelationExistsError returns a new ErrorTypeRelationExists.
func NewRelationExistsError(relationName string, schemaName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeRelationExists,
		Content: fmt.Sprintf("Relation %q already exists in schema %q", relationName, schemaName),
	}
}

// NewColumnNotExistsError returns a new ErrorTypeColumnNotExists.
func NewColumnNotExistsError(tableName string, columnName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeColumnNotExists,
		Content: fmt.Sprintf("Column `%s` does not exist in table `%s`", columnName, tableName),
	}
}

// NewIndexNotExistsError returns a new ErrorTypeIndexNotExists.
func NewIndexNotExistsError(tableName string, indexName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeIndexNotExists,
		Content: fmt.Sprintf("Index `%s` does not exist in table `%s`", indexName, tableName),
	}
}

// NewIndexExistsError returns a new ErrorTypeIndexExists.
func NewIndexExistsError(tableName string, indexName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeIndexExists,
		Content: fmt.Sprintf("Index `%s` already exists in table `%s`", indexName, tableName),
	}
}

// NewAccessOtherDatabaseError returns a new ErrorTypeAccessOtherDatabase.
func NewAccessOtherDatabaseError(current string, target string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeAccessOtherDatabase,
		Content: fmt.Sprintf("Database `%s` is not the current database `%s`", target, current),
	}
}

// NewTableNotExistsError returns a new ErrorTypeTableNotExists.
func NewTableNotExistsError(tableName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeTableNotExists,
		Content: fmt.Sprintf("Table `%s` does not exist", tableName),
	}
}

// NewTableExistsError returns a new ErrorTypeTableExists.
func NewTableExistsError(tableName string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeTableExists,
		Content: fmt.Sprintf("Table `%s` already exists", tableName),
	}
}

// Error implements the error interface.
func (e *WalkThroughError) Error() string {
	return e.Content
}

// WalkThrough will collect the catalog schema in the databaseState as it walks through the stmt.
func (d *DatabaseState) WalkThrough(stmt string) error {
	switch d.dbType {
	// only for mysqlwip test.
	case storepb.Engine_ENGINE_UNSPECIFIED:
		err := d.mysqlWalkThrough(stmt)
		return err
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		err := d.mysqlWalkThrough(stmt)
		return err
	case storepb.Engine_POSTGRES:
		if err := d.pgWalkThrough(stmt); err != nil {
			if d.ctx.CheckIntegrity {
				return err
			}
			// For PostgreSQL, we only support to walk-through with CheckIntegrity == true.
			// If CheckIntegrity == false, we'll try to walk-through with empty public schema.
			// We use `usable` to check if walk-through successfully.
			d.usable = false
		}
		return nil
	default:
		return &WalkThroughError{
			Type:    ErrorTypeUnsupported,
			Content: fmt.Sprintf("Walk-through doesn't support engine type: %s", d.dbType),
		}
	}
}

func (d *DatabaseState) mysqlWalkThrough(stmt string) error {
	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if _, exists := d.schemaSet[""]; !exists {
		d.createSchema("")
	}

	nodeList, err := d.parse(stmt)
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		// change state
		if err := d.changeState(node); err != nil {
			return err
		}
	}

	return nil
}

// compareIdentifier returns true if the engine will regard the two identifiers as the same one.
// TODO(zp): It's used for MySQL, we should refactor the package to make it more clear.
func compareIdentifier(a, b string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// isCurrentDatabase returns true if the given database is the current database of the state.
func (d *DatabaseState) isCurrentDatabase(database string) bool {
	return compareIdentifier(d.name, database, d.ctx.IgnoreCaseSensitive)
}

// isTableExists returns true if the given table exists in the database.
// TODO(zp): It's used for MySQL, we should refactor the package to make it more clear.
func (s *SchemaState) getTable(table string) (*TableState, bool) {
	for k, v := range s.tableSet {
		if compareIdentifier(k, table, s.ctx.IgnoreCaseSensitive) {
			return v, true
		}
	}

	return nil, false
}

func (d *DatabaseState) changeState(in tidbast.StmtNode) (err *WalkThroughError) {
	defer func() {
		if err == nil {
			return
		}
		if err.Line == 0 {
			err.Line = in.OriginTextPosition()
		}
	}()
	if d.deleted {
		return &WalkThroughError{
			Type:    ErrorTypeDatabaseIsDeleted,
			Content: fmt.Sprintf("Database `%s` is deleted", d.name),
		}
	}
	switch node := in.(type) {
	case *tidbast.CreateTableStmt:
		return d.createTable(node)
	case *tidbast.DropTableStmt:
		return d.dropTable(node)
	case *tidbast.AlterTableStmt:
		return d.alterTable(node)
	case *tidbast.CreateIndexStmt:
		return d.createIndex(node)
	case *tidbast.DropIndexStmt:
		return d.dropIndex(node)
	case *tidbast.AlterDatabaseStmt:
		return d.alterDatabase(node)
	case *tidbast.DropDatabaseStmt:
		return d.dropDatabase(node)
	case *tidbast.CreateDatabaseStmt:
		return NewAccessOtherDatabaseError(d.name, node.Name.O)
	case *tidbast.RenameTableStmt:
		return d.renameTable(node)
	default:
		return nil
	}
}

func (d *DatabaseState) renameTable(node *tidbast.RenameTableStmt) *WalkThroughError {
	for _, tableToTable := range node.TableToTables {
		schema, exists := d.schemaSet[""]
		if !exists {
			schema = d.createSchema("")
		}
		oldTableName := tableToTable.OldTable.Name.O
		newTableName := tableToTable.NewTable.Name.O
		if d.theCurrentDatabase(tableToTable) {
			if compareIdentifier(oldTableName, newTableName, d.ctx.IgnoreCaseSensitive) {
				return nil
			}
			table, exists := schema.getTable(oldTableName)
			if !exists {
				if schema.ctx.CheckIntegrity {
					return NewTableNotExistsError(oldTableName)
				}
				table = schema.createIncompleteTable(oldTableName)
			}
			if _, exists := schema.getTable(newTableName); exists {
				return NewTableExistsError(newTableName)
			}
			delete(schema.tableSet, table.name)
			table.name = newTableName
			schema.tableSet[table.name] = table
		} else if d.moveToOtherDatabase(tableToTable) {
			_, exists := schema.getTable(tableToTable.OldTable.Name.O)
			if !exists && schema.ctx.CheckIntegrity {
				return NewTableNotExistsError(tableToTable.OldTable.Name.O)
			}
			delete(schema.tableSet, tableToTable.OldTable.Name.O)
		} else {
			return NewAccessOtherDatabaseError(d.name, d.targetDatabase(tableToTable))
		}
	}
	return nil
}

func (d *DatabaseState) targetDatabase(node *tidbast.TableToTable) string {
	if node.OldTable.Schema.O != "" && !d.isCurrentDatabase(node.OldTable.Schema.O) {
		return node.OldTable.Schema.O
	}
	return node.NewTable.Schema.O
}

func (d *DatabaseState) moveToOtherDatabase(node *tidbast.TableToTable) bool {
	if node.OldTable.Schema.O != "" && !d.isCurrentDatabase(node.OldTable.Schema.O) {
		return false
	}
	return node.OldTable.Schema.O != node.NewTable.Schema.O
}

func (d *DatabaseState) theCurrentDatabase(node *tidbast.TableToTable) bool {
	if node.NewTable.Schema.O != "" && !d.isCurrentDatabase(node.NewTable.Schema.O) {
		return false
	}
	if node.OldTable.Schema.O != "" && !d.isCurrentDatabase(node.OldTable.Schema.O) {
		return false
	}
	return true
}

func (d *DatabaseState) dropDatabase(node *tidbast.DropDatabaseStmt) *WalkThroughError {
	if !d.isCurrentDatabase(node.Name.O) {
		return NewAccessOtherDatabaseError(d.name, node.Name.O)
	}

	d.deleted = true
	return nil
}

func (d *DatabaseState) alterDatabase(node *tidbast.AlterDatabaseStmt) *WalkThroughError {
	if !node.AlterDefaultDatabase && !d.isCurrentDatabase(node.Name.O) {
		return NewAccessOtherDatabaseError(d.name, node.Name.O)
	}

	for _, option := range node.Options {
		switch option.Tp {
		case tidbast.DatabaseOptionCharset:
			d.characterSet = option.Value
		case tidbast.DatabaseOptionCollate:
			d.collation = option.Value
		}
	}
	return nil
}

func (d *DatabaseState) findTableState(tableName *tidbast.TableName, createIncompleteTable bool) (*TableState, *WalkThroughError) {
	if tableName.Schema.O != "" && !d.isCurrentDatabase(tableName.Schema.O) {
		return nil, NewAccessOtherDatabaseError(d.name, tableName.Schema.O)
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema("")
	}

	table, exists := schema.getTable(tableName.Name.O)
	if !exists {
		if schema.ctx.CheckIntegrity {
			return nil, NewTableNotExistsError(tableName.Name.O)
		}
		if createIncompleteTable {
			table = schema.createIncompleteTable(tableName.Name.O)
		} else {
			return nil, nil
		}
	}

	return table, nil
}

func (d *DatabaseState) dropIndex(node *tidbast.DropIndexStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table, true /* createIncompleteTable */)
	if err != nil {
		return err
	}

	return table.dropIndex(d.ctx, node.IndexName)
}

func (d *DatabaseState) createIndex(node *tidbast.CreateIndexStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table, true /* createIncompleteTable */)
	if err != nil {
		return err
	}

	unique := false
	tp := model.IndexTypeBtree.String()
	isSpatial := false

	switch node.KeyType {
	case tidbast.IndexKeyTypeNone:
	case tidbast.IndexKeyTypeUnique:
		unique = true
	case tidbast.IndexKeyTypeFullText:
		tp = FullTextName
	case tidbast.IndexKeyTypeSpatial:
		isSpatial = true
		tp = SpatialName
	}

	keyList, err := table.validateAndGetKeyStringList(d.ctx, node.IndexPartSpecifications, false /* primary */, isSpatial)
	if err != nil {
		return err
	}

	return table.createIndex(node.IndexName, keyList, unique, tp, node.IndexOption)
}

func (d *DatabaseState) alterTable(node *tidbast.AlterTableStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table, true /* createIncompleteTable */)
	if err != nil {
		return err
	}

	for _, spec := range node.Specs {
		switch spec.Tp {
		case tidbast.AlterTableOption:
			for _, option := range spec.Options {
				switch option.Tp {
				case tidbast.TableOptionCollate:
					table.collation = newStringPointer(option.StrValue)
				case tidbast.TableOptionComment:
					table.comment = newStringPointer(option.StrValue)
				case tidbast.TableOptionEngine:
					table.engine = newStringPointer(option.StrValue)
				}
			}
		case tidbast.AlterTableAddColumns:
			for _, column := range spec.NewColumns {
				var pos *tidbast.ColumnPosition
				if len(spec.NewColumns) == 1 {
					pos = spec.Position
				}
				if err := table.createColumn(d.ctx, column, pos); err != nil {
					return err
				}
			}
			// MySQL can add table constraints in ALTER TABLE ADD COLUMN statements.
			for _, constraint := range spec.NewConstraints {
				if err := table.createConstraint(d.ctx, constraint); err != nil {
					return err
				}
			}
		case tidbast.AlterTableAddConstraint:
			if err := table.createConstraint(d.ctx, spec.Constraint); err != nil {
				return err
			}
		case tidbast.AlterTableDropColumn:
			if err := table.dropColumn(d.ctx, spec.OldColumnName.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableDropPrimaryKey:
			if err := table.dropIndex(d.ctx, PrimaryKeyName); err != nil {
				return err
			}
		case tidbast.AlterTableDropIndex:
			if err := table.dropIndex(d.ctx, spec.Name); err != nil {
				return err
			}
		case tidbast.AlterTableDropForeignKey:
			// we do not deal with DROP FOREIGN KEY statements.
		case tidbast.AlterTableModifyColumn:
			if err := table.changeColumn(d.ctx, spec.NewColumns[0].Name.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableChangeColumn:
			if err := table.changeColumn(d.ctx, spec.OldColumnName.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableRenameColumn:
			if err := table.renameColumn(d.ctx, spec.OldColumnName.Name.O, spec.NewColumnName.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableRenameTable:
			schema := d.schemaSet[""]
			if err := schema.renameTable(d.ctx, table.name, spec.NewTable.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableAlterColumn:
			if err := table.changeColumnDefault(d.ctx, spec.NewColumns[0]); err != nil {
				return err
			}
		case tidbast.AlterTableRenameIndex:
			if err := table.renameIndex(d.ctx, spec.FromKey.O, spec.ToKey.O); err != nil {
				return err
			}
		case tidbast.AlterTableIndexInvisible:
			if err := table.changeIndexVisibility(d.ctx, spec.IndexName.O, spec.Visibility); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *TableState) changeIndexVisibility(ctx *FinderContext, indexName string, visibility tidbast.IndexVisibility) *WalkThroughError {
	index, exists := t.indexSet[indexName]
	if !exists {
		if ctx.CheckIntegrity {
			return NewIndexNotExistsError(t.name, indexName)
		}
		index = t.createIncompleteIndex(indexName)
	}
	switch visibility {
	case tidbast.IndexVisibilityVisible:
		index.visible = newTruePointer()
	case tidbast.IndexVisibilityInvisible:
		index.visible = newFalsePointer()
	}
	return nil
}

func (t *TableState) renameIndex(ctx *FinderContext, oldName string, newName string) *WalkThroughError {
	// For MySQL, the primary key has a special name 'PRIMARY'.
	// And the other indexes cannot use the name which case-insensitive equals 'PRIMARY'.
	if strings.ToUpper(oldName) == PrimaryKeyName || strings.ToUpper(newName) == PrimaryKeyName {
		incorrectName := oldName
		if strings.ToUpper(oldName) != PrimaryKeyName {
			incorrectName = newName
		}
		return &WalkThroughError{
			Type:    ErrorTypeIncorrectIndexName,
			Content: fmt.Sprintf("Incorrect index name `%s`", incorrectName),
		}
	}

	index, exists := t.indexSet[oldName]
	if !exists {
		if ctx.CheckIntegrity {
			return NewIndexNotExistsError(t.name, oldName)
		}
		index = t.createIncompleteIndex(oldName)
	}

	if _, exists := t.indexSet[newName]; exists {
		return NewIndexExistsError(t.name, newName)
	}

	index.name = newName
	delete(t.indexSet, oldName)
	t.indexSet[newName] = index
	return nil
}

func (t *TableState) createIncompleteIndex(name string) *IndexState {
	index := &IndexState{
		name: name,
	}
	t.indexSet[name] = index
	return index
}

func (t *TableState) changeColumnDefault(ctx *FinderContext, column *tidbast.ColumnDef) *WalkThroughError {
	columnName := column.Name.Name.O
	colState, exists := t.columnSet[columnName]
	if !exists {
		if ctx.CheckIntegrity {
			return NewColumnNotExistsError(t.name, columnName)
		}
		colState = t.createIncompleteColumn(columnName)
	}

	if len(column.Options) == 1 {
		// SET DEFAULT
		if column.Options[0].Expr.GetType().GetType() != mysql.TypeNull {
			if colState.columnType != nil {
				switch strings.ToLower(*colState.columnType) {
				case "blob", "tinyblob", "mediumblob", "longblob",
					"text", "tinytext", "mediumtext", "longtext",
					"json",
					"geometry":
					return &WalkThroughError{
						Type: ErrorTypeInvalidColumnTypeForDefaultValue,
						// Content comes from MySQL Error content.
						Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
					}
				}
			}

			defaultValue, err := restoreNode(column.Options[0].Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			colState.defaultValue = &defaultValue
		} else {
			if colState.nullable != nil && !*colState.nullable {
				return &WalkThroughError{
					Type: ErrorTypeSetNullDefaultForNotNullColumn,
					// Content comes from MySQL Error content.
					Content: fmt.Sprintf("Invalid default value for column `%s`", columnName),
				}
			}
			colState.defaultValue = nil
		}
	} else {
		// DROP DEFAULT
		colState.defaultValue = nil
	}
	return nil
}

func (s *SchemaState) renameTable(ctx *FinderContext, oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	table, exists := s.getTable(oldName)
	if !exists {
		if ctx.CheckIntegrity {
			return &WalkThroughError{
				Type:    ErrorTypeTableNotExists,
				Content: fmt.Sprintf("Table `%s` does not exist", oldName),
			}
		}
		table = s.createIncompleteTable(oldName)
	}

	if _, exists := s.getTable(newName); exists {
		return &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf("Table `%s` already exists", newName),
		}
	}

	table.name = newName
	delete(s.tableSet, oldName)
	s.tableSet[newName] = table
	return nil
}

func (s *SchemaState) createIncompleteTable(name string) *TableState {
	table := &TableState{
		name:      name,
		columnSet: make(columnStateMap),
		indexSet:  make(IndexStateMap),
	}
	s.tableSet[name] = table
	return table
}

func (t *TableState) renameColumn(ctx *FinderContext, oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	column, exists := t.columnSet[oldName]
	if !exists {
		if ctx.CheckIntegrity {
			return &WalkThroughError{
				Type:    ErrorTypeColumnNotExists,
				Content: fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, t.name),
			}
		}
		column = t.createIncompleteColumn(oldName)
	}

	if _, exists := t.columnSet[newName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s", newName, t.name),
		}
	}

	column.name = newName
	delete(t.columnSet, oldName)
	t.columnSet[newName] = column

	t.renameColumnInIndexKey(oldName, newName)
	return nil
}

func (t *TableState) createIncompleteColumn(name string) *ColumnState {
	column := &ColumnState{
		name: name,
	}
	t.columnSet[name] = column
	return column
}

func (t *TableState) renameColumnInIndexKey(oldName string, newName string) {
	if oldName == newName {
		return
	}
	for _, index := range t.indexSet {
		for i, key := range index.expressionList {
			if key == oldName {
				index.expressionList[i] = newName
			}
		}
	}
}

// completeTableChangeColumn changes column definition.
// It works as:
// 1. drop column from tableState.columnSet, but do not drop column from indexSet.
// 2. rename column from indexSet.
// 3. create a new column in columnSet.
func (t *TableState) completeTableChangeColumn(ctx *FinderContext, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	column, exists := t.columnSet[oldName]
	if !exists {
		return NewColumnNotExistsError(t.name, oldName)
	}

	pos := *column.position

	// generate Position struct for creating new column
	if position == nil {
		position = &tidbast.ColumnPosition{Tp: tidbast.ColumnPositionNone}
	}
	if position.Tp == tidbast.ColumnPositionNone {
		if pos == 1 {
			position.Tp = tidbast.ColumnPositionFirst
		} else {
			for _, col := range t.columnSet {
				if *col.position == pos-1 {
					position.Tp = tidbast.ColumnPositionAfter
					position.RelativeColumn = &tidbast.ColumnName{Name: model.NewCIStr(col.name)}
					break
				}
			}
		}
	}

	// drop column from columnSet
	for _, col := range t.columnSet {
		if *col.position > pos {
			*col.position--
		}
	}
	delete(t.columnSet, column.name)

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(ctx, newColumn, position)
}

// incompleteTableChangeColumn changes column definition.
// It does not maintain the position of the column.
func (t *TableState) incompleteTableChangeColumn(ctx *FinderContext, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	delete(t.columnSet, oldName)

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(ctx, newColumn, position)
}

func (t *TableState) changeColumn(ctx *FinderContext, oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	if ctx.CheckIntegrity {
		return t.completeTableChangeColumn(ctx, oldName, newColumn, position)
	}
	return t.incompleteTableChangeColumn(ctx, oldName, newColumn, position)
}

func (t *TableState) dropIndex(ctx *FinderContext, indexName string) *WalkThroughError {
	if ctx.CheckIntegrity {
		if _, exists := t.indexSet[indexName]; !exists {
			if indexName == PrimaryKeyName {
				return &WalkThroughError{
					Type:    ErrorTypePrimaryKeyNotExists,
					Content: fmt.Sprintf("Primary key does not exist in table `%s`", t.name),
				}
			}
			return NewIndexNotExistsError(t.name, indexName)
		}
	}

	delete(t.indexSet, indexName)
	return nil
}

func (t *TableState) completeTableDropColumn(columnName string) *WalkThroughError {
	column, exists := t.columnSet[columnName]
	if !exists {
		return NewColumnNotExistsError(t.name, columnName)
	}

	// Cannot drop all columns in a table using ALTER TABLE DROP COLUMN.
	if len(t.columnSet) == 1 {
		return &WalkThroughError{
			Type: ErrorTypeDropAllColumns,
			// Error content comes from MySQL error content.
			Content: fmt.Sprintf("Can't delete all columns with ALTER TABLE; use DROP TABLE %s instead", t.name),
		}
	}

	// If columns are dropped from a table, the columns are also removed from any index of which they are a part.
	for _, index := range t.indexSet {
		index.dropColumn(columnName)
		// If all columns that make up an index are dropped, the index is dropped as well.
		if len(index.expressionList) == 0 {
			delete(t.indexSet, index.name)
		}
	}

	// modify the column position
	for _, col := range t.columnSet {
		if *col.position > *column.position {
			*col.position--
		}
	}

	delete(t.columnSet, columnName)
	return nil
}

func (t *TableState) incompleteTableDropColumn(columnName string) *WalkThroughError {
	// If columns are dropped from a table, the columns are also removed from any index of which they are a part.
	for _, index := range t.indexSet {
		if len(index.expressionList) == 0 {
			continue
		}
		index.dropColumn(columnName)
		// If all columns that make up an index are dropped, the index is dropped as well.
		if len(index.expressionList) == 0 {
			delete(t.indexSet, index.name)
		}
	}

	delete(t.columnSet, columnName)
	return nil
}

func (t *TableState) dropColumn(ctx *FinderContext, columnName string) *WalkThroughError {
	if ctx.CheckIntegrity {
		return t.completeTableDropColumn(columnName)
	}
	return t.incompleteTableDropColumn(columnName)
}

func (idx *IndexState) dropColumn(columnName string) {
	if len(idx.expressionList) == 0 {
		return
	}
	var newKeyList []string
	for _, key := range idx.expressionList {
		if key != columnName {
			newKeyList = append(newKeyList, key)
		}
	}

	idx.expressionList = newKeyList
}

// reorderColumn reorders the columns for new column and returns the new column position.
func (t *TableState) reorderColumn(position *tidbast.ColumnPosition) (int, *WalkThroughError) {
	switch position.Tp {
	case tidbast.ColumnPositionNone:
		return len(t.columnSet) + 1, nil
	case tidbast.ColumnPositionFirst:
		for _, column := range t.columnSet {
			*column.position++
		}
		return 1, nil
	case tidbast.ColumnPositionAfter:
		columnName := position.RelativeColumn.Name.O
		column, exist := t.columnSet[columnName]
		if !exist {
			return 0, NewColumnNotExistsError(t.name, columnName)
		}
		for _, col := range t.columnSet {
			if *col.position > *column.position {
				*col.position++
			}
		}
		return *column.position + 1, nil
	}
	return 0, &WalkThroughError{
		Type:    ErrorTypeUnsupported,
		Content: fmt.Sprintf("Unsupported column position type: %d", position.Tp),
	}
}

func (d *DatabaseState) dropTable(node *tidbast.DropTableStmt) *WalkThroughError {
	// TODO(rebelice): deal with DROP VIEW statement.
	if !node.IsView {
		for _, name := range node.Tables {
			if name.Schema.O != "" && !d.isCurrentDatabase(name.Schema.O) {
				return &WalkThroughError{
					Type:    ErrorTypeAccessOtherDatabase,
					Content: fmt.Sprintf("Database `%s` is not the current database `%s`", name.Schema.O, d.name),
				}
			}

			schema, exists := d.schemaSet[""]
			if !exists {
				schema = d.createSchema("")
			}

			table, exists := schema.getTable(name.Name.O)
			if !exists {
				if node.IfExists || !d.ctx.CheckIntegrity {
					return nil
				}
				return &WalkThroughError{
					Type:    ErrorTypeTableNotExists,
					Content: fmt.Sprintf("Table `%s` does not exist", name.Name.O),
				}
			}

			delete(schema.tableSet, table.name)
		}
	}
	return nil
}

func (d *DatabaseState) copyTable(node *tidbast.CreateTableStmt) *WalkThroughError {
	targetTable, err := d.findTableState(node.ReferTable, true /* createIncompleteTable */)
	if err != nil {
		return err
	}

	schema := d.schemaSet[""]
	table := targetTable.copy()
	table.name = node.Table.Name.O
	schema.tableSet[table.name] = table
	return nil
}

func (d *DatabaseState) createTable(node *tidbast.CreateTableStmt) *WalkThroughError {
	if node.Table.Schema.O != "" && !d.isCurrentDatabase(node.Table.Schema.O) {
		return &WalkThroughError{
			Type:    ErrorTypeAccessOtherDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", node.Table.Schema.O, d.name),
		}
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema("")
	}

	if _, exists = schema.getTable(node.Table.Name.O); exists {
		if node.IfNotExists {
			return nil
		}
		return &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf("Table `%s` already exists", node.Table.Name.O),
		}
	}

	if node.Select != nil {
		return &WalkThroughError{
			Type:    ErrorTypeUseCreateTableAs,
			Content: fmt.Sprintf("Disallow the CREATE TABLE AS statement but \"%s\" uses", node.Text()),
		}
	}

	if node.ReferTable != nil {
		return d.copyTable(node)
	}

	table := &TableState{
		name:      node.Table.Name.O,
		engine:    newEmptyStringPointer(),
		collation: newEmptyStringPointer(),
		comment:   newEmptyStringPointer(),
		columnSet: make(columnStateMap),
		indexSet:  make(IndexStateMap),
	}
	schema.tableSet[table.name] = table
	hasAutoIncrement := false

	for _, column := range node.Cols {
		if isAutoIncrement(column) {
			if hasAutoIncrement {
				return &WalkThroughError{
					Type: ErrorTypeAutoIncrementExists,
					// The content comes from MySQL error content.
					Content: fmt.Sprintf("There can be only one auto column for table `%s`", table.name),
				}
			}
			hasAutoIncrement = true
		}
		if err := table.createColumn(d.ctx, column, nil /* position */); err != nil {
			err.Line = column.OriginTextPosition()
			return err
		}
	}

	for _, constraint := range node.Constraints {
		if err := table.createConstraint(d.ctx, constraint); err != nil {
			err.Line = constraint.OriginTextPosition()
			return err
		}
	}

	return nil
}

func (t *TableState) createConstraint(ctx *FinderContext, constraint *tidbast.Constraint) *WalkThroughError {
	switch constraint.Tp {
	case tidbast.ConstraintPrimaryKey:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, true /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createPrimaryKey(keyList, getIndexType(constraint.Option)); err != nil {
			return err
		}
	case tidbast.ConstraintKey, tidbast.ConstraintIndex:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, false /* unique */, getIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, true /* unique */, getIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintForeignKey:
		// we do not deal with FOREIGN KEY constraints
	case tidbast.ConstraintFulltext:
		keyList, err := t.validateAndGetKeyStringList(ctx, constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, false /* unique */, FullTextName, constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintCheck:
		// we do not deal with CHECK constraints
	}

	return nil
}

func (t *TableState) validateAndGetKeyStringList(ctx *FinderContext, keyList []*tidbast.IndexPartSpecification, primary bool, isSpatial bool) ([]string, *WalkThroughError) {
	var res []string
	for _, key := range keyList {
		if key.Expr != nil {
			str, err := restoreNode(key, format.DefaultRestoreFlags)
			if err != nil {
				return nil, err
			}
			res = append(res, str)
		} else {
			columnName := key.Column.Name.O
			column, exists := t.columnSet[columnName]
			if !exists {
				if ctx.CheckIntegrity {
					return nil, NewColumnNotExistsError(t.name, columnName)
				}
			} else {
				if primary {
					column.nullable = newFalsePointer()
				}
				if isSpatial && column.nullable != nil && *column.nullable {
					return nil, &WalkThroughError{
						Type: ErrorTypeSpatialIndexKeyNullable,
						// The error content comes from MySQL.
						Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name),
					}
				}
			}

			res = append(res, columnName)
		}
	}
	return res, nil
}

func isAutoIncrement(column *tidbast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionAutoIncrement {
			return true
		}
	}
	return false
}

func checkDefault(columnName string, columnType *types.FieldType, value tidbast.ExprNode) *WalkThroughError {
	if value.GetType().GetType() != mysql.TypeNull {
		switch columnType.GetType() {
		case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeJSON, mysql.TypeGeometry:
			return &WalkThroughError{
				Type: ErrorTypeInvalidColumnTypeForDefaultValue,
				// Content comes from MySQL Error content.
				Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
			}
		}
	}

	if valueExpr, yes := value.(tidbast.ValueExpr); yes {
		datum := types.NewDatum(valueExpr.GetValue())
		if _, err := datum.ConvertTo(&stmtctx.StatementContext{}, columnType); err != nil {
			return &WalkThroughError{
				Type:    ErrorTypeInvalidColumnTypeForDefaultValue,
				Content: err.Error(),
			}
		}
	}
	return nil
}

func (t *TableState) createColumn(ctx *FinderContext, column *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	if _, exists := t.columnSet[column.Name.Name.O]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", column.Name.Name.O, t.name),
		}
	}

	pos := len(t.columnSet) + 1
	if position != nil && ctx.CheckIntegrity {
		var err *WalkThroughError
		pos, err = t.reorderColumn(position)
		if err != nil {
			return err
		}
	}

	vTrue := true
	col := &ColumnState{
		name:         column.Name.Name.O,
		position:     &pos,
		defaultValue: nil,
		nullable:     &vTrue,
		columnType:   newStringPointer(column.Tp.CompactStr()),
		characterSet: newStringPointer(column.Tp.GetCharset()),
		collation:    newStringPointer(column.Tp.GetCollate()),
		comment:      newEmptyStringPointer(),
	}
	setNullDefault := false

	for _, option := range column.Options {
		switch option.Tp {
		case tidbast.ColumnOptionPrimaryKey:
			col.nullable = newFalsePointer()
			if err := t.createPrimaryKey([]string{col.name}, model.IndexTypeBtree.String()); err != nil {
				return err
			}
		case tidbast.ColumnOptionNotNull:
			col.nullable = newFalsePointer()
		case tidbast.ColumnOptionAutoIncrement:
			// we do not deal with AUTO-INCREMENT
		case tidbast.ColumnOptionDefaultValue:
			if err := checkDefault(col.name, column.Tp, option.Expr); err != nil {
				return err
			}
			if option.Expr.GetType().GetType() != mysql.TypeNull {
				defaultValue, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
				if err != nil {
					return err
				}
				col.defaultValue = &defaultValue
			} else {
				setNullDefault = true
			}
		case tidbast.ColumnOptionUniqKey:
			if err := t.createIndex("", []string{col.name}, true /* unique */, model.IndexTypeBtree.String(), nil); err != nil {
				return err
			}
		case tidbast.ColumnOptionNull:
			col.nullable = newTruePointer()
		case tidbast.ColumnOptionOnUpdate:
			// we do not deal with ON UPDATE
			if column.Tp.GetType() != mysql.TypeDatetime && column.Tp.GetType() != mysql.TypeTimestamp {
				return &WalkThroughError{
					Type:    ErrorTypeOnUpdateColumnNotDatetimeOrTimestamp,
					Content: fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.name),
				}
			}
		case tidbast.ColumnOptionComment:
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.comment = &comment
		case tidbast.ColumnOptionGenerated:
			// we do not deal with GENERATED ALWAYS AS
		case tidbast.ColumnOptionReference:
			// MySQL will ignore the inline REFERENCE
			// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		case tidbast.ColumnOptionCollate:
			col.collation = newStringPointer(option.StrValue)
		case tidbast.ColumnOptionCheck:
			// we do not deal with CHECK constraint
		case tidbast.ColumnOptionColumnFormat:
			// we do not deal with COLUMN_FORMAT
		case tidbast.ColumnOptionStorage:
			// we do not deal with STORAGE
		case tidbast.ColumnOptionAutoRandom:
			// we do not deal with AUTO_RANDOM
		}
	}

	if col.nullable != nil && !*col.nullable && setNullDefault {
		return &WalkThroughError{
			Type: ErrorTypeSetNullDefaultForNotNullColumn,
			// Content comes from MySQL Error content.
			Content: fmt.Sprintf("Invalid default value for column `%s`", col.name),
		}
	}

	t.columnSet[col.name] = col
	return nil
}

func (t *TableState) createIndex(name string, keyList []string, unique bool, tp string, option *tidbast.IndexOption) *WalkThroughError {
	if len(keyList) == 0 {
		return &WalkThroughError{
			Type:    ErrorTypeIndexEmptyKeys,
			Content: fmt.Sprintf("Index `%s` in table `%s` has empty key", name, t.name),
		}
	}
	if name != "" {
		if _, exists := t.indexSet[name]; exists {
			return NewIndexExistsError(t.name, name)
		}
	} else {
		suffix := 1
		for {
			name = keyList[0]
			if suffix > 1 {
				name = fmt.Sprintf("%s_%d", keyList[0], suffix)
			}
			if _, exists := t.indexSet[name]; !exists {
				break
			}
			suffix++
		}
	}

	index := &IndexState{
		name:           name,
		expressionList: keyList,
		indexType:      &tp,
		unique:         &unique,
		primary:        newFalsePointer(),
		visible:        newTruePointer(),
		comment:        newEmptyStringPointer(),
	}

	if option != nil && option.Visibility == tidbast.IndexVisibilityInvisible {
		index.visible = newFalsePointer()
	}

	t.indexSet[name] = index
	return nil
}

func (t *TableState) createPrimaryKey(keys []string, tp string) *WalkThroughError {
	if _, exists := t.indexSet[PrimaryKeyName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypePrimaryKeyExists,
			Content: fmt.Sprintf("Primary key exists in table `%s`", t.name),
		}
	}

	pk := &IndexState{
		name:           PrimaryKeyName,
		expressionList: keys,
		indexType:      &tp,
		unique:         newTruePointer(),
		primary:        newTruePointer(),
		visible:        newTruePointer(),
		comment:        newEmptyStringPointer(),
	}
	t.indexSet[pk.name] = pk
	return nil
}

func (d *DatabaseState) createSchema(name string) *SchemaState {
	schema := &SchemaState{
		ctx:      d.ctx.Copy(),
		name:     name,
		tableSet: make(tableStateMap),
		viewSet:  make(viewStateMap),
	}

	d.schemaSet[name] = schema
	return schema
}

func (*DatabaseState) parse(statement string) ([]tidbast.StmtNode, *WalkThroughError) {
	p := tidbparser.New()
	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	treeList, err := mysqlbbparser.ParseMySQL(statement)
	if err != nil {
		return nil, NewParseError(err.Error())
	}
	if len(treeList) == 0 {
		return nil, nil
	}

	var returnNodes []tidbast.StmtNode
	for _, item := range treeList {
		tree := item.Tree
		tokens := item.Tokens

		for _, child := range tree.GetChildren() {
			if child == nil {
				continue
			}

			if query, ok := child.(mysqlparser.IQueryContext); ok {
				text := tokens.GetTextFromRuleContext(query)
				lastLine := query.GetStop().GetLine() + item.BaseLine

				if nodes, _, err := p.Parse(text, "", ""); err == nil {
					if len(nodes) != 1 {
						continue
					}
					node := nodes[0]
					node.SetText(nil, text)
					node.SetOriginTextPosition(lastLine)
					if n, ok := node.(*tidbast.CreateTableStmt); ok {
						if err := tidbbbparser.SetLineForMySQLCreateTableStmt(n); err != nil {
							return nil, NewParseError(err.Error())
						}
					}
					returnNodes = append(returnNodes, node)
				}
			}
		}
	}

	return returnNodes, nil
}

func restoreNode(node tidbast.Node, flag format.RestoreFlags) (string, *WalkThroughError) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", &WalkThroughError{
			Type:    ErrorTypeDeparseError,
			Content: err.Error(),
		}
	}
	return buffer.String(), nil
}

func getIndexType(option *tidbast.IndexOption) string {
	if option != nil {
		switch option.Tp {
		case model.IndexTypeBtree,
			model.IndexTypeHash,
			model.IndexTypeRtree:
			return option.Tp.String()
		}
	}
	return model.IndexTypeBtree.String()
}
