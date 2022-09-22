package catalog

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/parser"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
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

	// 101 parse error type.

	// ErrorTypeParseError is the error in parsing.
	ErrorTypeParseError WalkThroughErrorType = 101
	// ErrorTypeRestoreError is the error in restoring.
	ErrorTypeRestoreError WalkThroughErrorType = 102

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

	// 401 ~ 499 column error type.

	// ErrorTypeColumnExists is the error that column exists.
	ErrorTypeColumnExists = 401
	// ErrorTypeColumnNotExists is the error that column not exists.
	ErrorTypeColumnNotExists = 402
	// ErrorTypeDropAllColumns is the error that dropping all columns in a table.
	ErrorTypeDropAllColumns = 403

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
)

// WalkThroughError is the error for walking-through.
type WalkThroughError struct {
	Type    WalkThroughErrorType
	Content string
	Line    int
}

// NewParseError returns a new ErrorTypeParseError.
func NewParseError(content string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeParseError,
		Content: content,
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

// WalkThrough will collect the catalog schema in the databaseState as it walks through the stmts.
func (d *DatabaseState) WalkThrough(stmts string) error {
	if d.dbType != db.MySQL && d.dbType != db.TiDB {
		return &WalkThroughError{
			Type:    ErrorTypeUnsupported,
			Content: fmt.Sprintf("Walk-through doesn't support engine type: %s", d.dbType),
		}
	}

	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if _, exists := d.schemaSet[""]; !exists {
		d.createSchema("")
	}

	nodeList, err := d.parse(stmts)
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		// validate DML
		if dml, ok := node.(tidbast.DMLNode); ok {
			if err := d.validateDML(dml); err != nil {
				return err
			}
			continue
		}
		// change state
		if err := d.changeState(node); err != nil {
			return err
		}
	}

	return nil
}

func (d *DatabaseState) validateDML(in tidbast.DMLNode) (err *WalkThroughError) {
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
	case *tidbast.InsertStmt:
		return d.checkInsert(node)
	default:
		return nil
	}
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
		return NewAccessOtherDatabaseError(d.name, node.Name)
	case *tidbast.RenameTableStmt:
		return d.renameTable(node)
	default:
		return nil
	}
}

func (d *DatabaseState) checkInsert(node *tidbast.InsertStmt) *WalkThroughError {
	tableName, ok := getTableSourceName(node.Table)
	if ok {
		table, err := d.findTableState(tableName)
		if err != nil {
			return err
		}
		if table == nil {
			return nil
		}
		for _, col := range node.Columns {
			if _, exists := table.columnSet[col.Name.O]; !exists {
				return NewColumnNotExistsError(table.name, col.Name.O)
			}
		}
	}
	return nil
}

func (d *DatabaseState) renameTable(node *tidbast.RenameTableStmt) *WalkThroughError {
	for _, tableToTable := range node.TableToTables {
		schema, exists := d.schemaSet[""]
		if !exists {
			schema = d.createSchema("")
		}
		if d.theCurrentDatabase(tableToTable) {
			table, exists := schema.tableSet[tableToTable.OldTable.Name.O]
			if !exists {
				return NewTableNotExistsError(tableToTable.OldTable.Name.O)
			}
			if _, exists := schema.tableSet[tableToTable.NewTable.Name.O]; exists {
				return NewTableExistsError(tableToTable.NewTable.Name.O)
			}
			delete(schema.tableSet, table.name)
			table.name = tableToTable.NewTable.Name.O
			schema.tableSet[table.name] = table
		} else if d.moveToOtherDatabase(tableToTable) {
			_, exists := schema.tableSet[tableToTable.OldTable.Name.O]
			if !exists {
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
	if node.OldTable.Schema.O != "" && node.OldTable.Schema.O != d.name {
		return node.OldTable.Schema.O
	}
	return node.NewTable.Schema.O
}

func (d *DatabaseState) moveToOtherDatabase(node *tidbast.TableToTable) bool {
	if node.OldTable.Schema.O != "" && node.OldTable.Schema.O != d.name {
		return false
	}
	return node.OldTable.Schema.O != node.NewTable.Schema.O
}

func (d *DatabaseState) theCurrentDatabase(node *tidbast.TableToTable) bool {
	if node.NewTable.Schema.O != "" && node.NewTable.Schema.O != d.name {
		return false
	}
	if node.OldTable.Schema.O != "" && node.OldTable.Schema.O != d.name {
		return false
	}
	return true
}

func (d *DatabaseState) dropDatabase(node *tidbast.DropDatabaseStmt) *WalkThroughError {
	if d.name != "" && node.Name != d.name {
		return NewAccessOtherDatabaseError(d.name, node.Name)
	}

	d.deleted = true
	return nil
}

func (d *DatabaseState) alterDatabase(node *tidbast.AlterDatabaseStmt) *WalkThroughError {
	if !node.AlterDefaultDatabase && d.name != "" && node.Name != d.name {
		return NewAccessOtherDatabaseError(d.name, node.Name)
	}

	for _, option := range node.Options {
		switch option.Tp {
		case tidbast.DatabaseOptionCharset:
			d.characterSet = newStateString(option.Value)
		case tidbast.DatabaseOptionCollate:
			d.collation = newStateString(option.Value)
		}
	}
	return nil
}

func (d *DatabaseState) findTableState(tableName *tidbast.TableName) (*TableState, *WalkThroughError) {
	if tableName.Schema.O != "" && d.name != "" && tableName.Schema.O != d.name {
		return nil, NewAccessOtherDatabaseError(d.name, tableName.Schema.O)
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema("")
	}

	table, exists := schema.tableSet[tableName.Name.O]
	if !exists {
		if schema.complete {
			return nil, NewTableNotExistsError(tableName.Name.O)
		}
		table = schema.createIncompleteTable(tableName.Name.O)
	}

	return table, nil
}

func (d *DatabaseState) dropIndex(node *tidbast.DropIndexStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}
	if table == nil {
		return nil
	}

	return table.dropIndex(node.IndexName)
}

func (d *DatabaseState) createIndex(node *tidbast.CreateIndexStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}
	if table == nil {
		return nil
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

	keyList, err := table.validateAndGetKeyStringList(node.IndexPartSpecifications, false /* primary */, isSpatial)
	if err != nil {
		return err
	}

	return table.createIndex(node.IndexName, keyList, unique, tp, node.IndexOption)
}

func (d *DatabaseState) alterTable(node *tidbast.AlterTableStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}

	for _, spec := range node.Specs {
		switch spec.Tp {
		case tidbast.AlterTableOption:
			for _, option := range spec.Options {
				switch option.Tp {
				case tidbast.TableOptionCollate:
					table.collation = newStateString(option.StrValue)
				case tidbast.TableOptionComment:
					table.comment = newStateString(option.StrValue)
				case tidbast.TableOptionEngine:
					table.engine = newStateString(option.StrValue)
				}
			}
		case tidbast.AlterTableAddColumns:
			for _, column := range spec.NewColumns {
				var pos *tidbast.ColumnPosition
				if len(spec.NewColumns) == 1 {
					pos = spec.Position
				}
				if err := table.createColumn(column, pos); err != nil {
					return err
				}
			}
			// MySQL can add table constraints in ALTER TABLE ADD COLUMN statements.
			for _, constraint := range spec.NewConstraints {
				if err := table.createConstraint(constraint); err != nil {
					return err
				}
			}
		case tidbast.AlterTableAddConstraint:
			if err := table.createConstraint(spec.Constraint); err != nil {
				return err
			}
		case tidbast.AlterTableDropColumn:
			if err := table.dropColumn(spec.OldColumnName.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableDropPrimaryKey:
			if err := table.dropIndex(PrimaryKeyName); err != nil {
				return err
			}
		case tidbast.AlterTableDropIndex:
			if err := table.dropIndex(spec.Name); err != nil {
				return err
			}
		case tidbast.AlterTableDropForeignKey:
			// we do not deal with DROP FOREIGN KEY statements.
		case tidbast.AlterTableModifyColumn:
			if err := table.changeColumn(spec.NewColumns[0].Name.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableChangeColumn:
			if err := table.changeColumn(spec.OldColumnName.Name.O, spec.NewColumns[0], spec.Position); err != nil {
				return err
			}
		case tidbast.AlterTableRenameColumn:
			if err := table.renameColumn(spec.OldColumnName.Name.O, spec.NewColumnName.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableRenameTable:
			schema := d.schemaSet[""]
			if err := schema.renameTable(table.name, spec.NewTable.Name.O); err != nil {
				return err
			}
		case tidbast.AlterTableAlterColumn:
			if err := table.changeColumnDefault(spec.NewColumns[0]); err != nil {
				return err
			}
		case tidbast.AlterTableRenameIndex:
			if err := table.renameIndex(spec.FromKey.O, spec.ToKey.O); err != nil {
				return err
			}
		case tidbast.AlterTableIndexInvisible:
			if err := table.changeIndexVisibility(spec.IndexName.O, spec.Visibility); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *TableState) changeIndexVisibility(indexName string, visibility tidbast.IndexVisibility) *WalkThroughError {
	index, exists := t.indexSet[indexName]
	if !exists {
		if t.complete {
			return NewIndexNotExistsError(t.name, indexName)
		}
		index = t.createIncompleteIndex(indexName)
	}
	switch visibility {
	case tidbast.IndexVisibilityVisible:
		index.visible = newStateBool(true)
	case tidbast.IndexVisibilityInvisible:
		index.visible = newStateBool(false)
	}
	return nil
}

func (t *TableState) renameIndex(oldName string, newName string) *WalkThroughError {
	// For MySQL, the primary key has a special name 'PRIMARY'.
	// And the other indexes can not use the name which case-insensitive equals 'PRIMARY'.
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
		if t.complete {
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
		complete: false,
		name:     name,
	}
	t.indexSet[name] = index
	return index
}

func (t *TableState) changeColumnDefault(column *tidbast.ColumnDef) *WalkThroughError {
	columnName := column.Name.Name.O
	colState, exists := t.columnSet[columnName]
	if !exists {
		if t.complete {
			return NewColumnNotExistsError(t.name, columnName)
		}
		colState = t.createIncompleteColumn(columnName)
	}

	if len(column.Options) == 1 {
		// SET DEFAULT
		defaultValue, err := restoreNode(column.Options[0].Expr, format.RestoreStringWithoutCharset)
		if err != nil {
			return err
		}
		colState.defaultValue = newStateStringPointer(&defaultValue)
	} else {
		// DROP DEFAULT
		colState.defaultValue = newStateStringPointer(nil)
	}
	return nil
}

func (s *SchemaState) completeSchemaRenameTable(oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	table, exists := s.tableSet[oldName]
	if !exists {
		return &WalkThroughError{
			Type:    ErrorTypeTableNotExists,
			Content: fmt.Sprintf("Table `%s` does not exist", oldName),
		}
	}

	if _, exists := s.tableSet[newName]; exists {
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

func (s *SchemaState) incompleteSchemaRenameTable(oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	if _, exists := s.tableSet[newName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf("Table `%s` already exists", newName),
		}
	}

	table, exists := s.tableSet[oldName]
	if exists {
		table.name = newName
		delete(s.tableSet, oldName)
		s.tableSet[newName] = table
	} else {
		s.createIncompleteTable(newName)
	}

	return nil
}

func (s *SchemaState) createIncompleteTable(name string) *TableState {
	table := &TableState{
		complete: false,
		name:     name,
	}
	s.tableSet[name] = table
	return table
}

func (s *SchemaState) renameTable(oldName string, newName string) *WalkThroughError {
	if s.complete {
		return s.completeSchemaRenameTable(oldName, newName)
	}
	return s.incompleteSchemaRenameTable(oldName, newName)
}

func (t *TableState) completeTableRenameColumn(oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	column, exists := t.columnSet[oldName]
	if !exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnNotExists,
			Content: fmt.Sprintf("Column `%s` does not exist in table `%s`", oldName, t.name),
		}
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

func (t *TableState) incompleteTableRenameColumn(oldName string, newName string) *WalkThroughError {
	if oldName == newName {
		return nil
	}

	if _, exists := t.columnSet[newName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s", newName, t.name),
		}
	}

	column, exists := t.columnSet[oldName]
	if exists {
		column.name = newName
		delete(t.columnSet, oldName)
		t.columnSet[newName] = column
	} else {
		t.createIncompleteColumn(newName)
	}

	t.renameColumnInIndexKey(oldName, newName)
	return nil
}

func (t *TableState) createIncompleteColumn(name string) *ColumnState {
	column := &ColumnState{
		complete: false,
		name:     name,
	}
	t.columnSet[name] = column
	return column
}

func (t *TableState) renameColumn(oldName string, newName string) *WalkThroughError {
	if t.complete {
		return t.completeTableRenameColumn(oldName, newName)
	}
	return t.incompleteTableRenameColumn(oldName, newName)
}

func (t *TableState) renameColumnInIndexKey(oldName string, newName string) {
	if oldName == newName {
		return
	}
	for _, index := range t.indexSet {
		if index.expressionList.defined {
			for i, key := range index.expressionList.value {
				if key == oldName {
					index.expressionList.value[i] = newName
				}
			}
		}
	}
}

// completeTableChangeColumn changes column definition.
// It works as:
// 1. drop column from tableState.columnSet, but do not drop column from indexSet.
// 2. rename column from indexSet.
// 3. create a new column in columnSet.
func (t *TableState) completeTableChangeColumn(oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	column, exists := t.columnSet[oldName]
	if !exists {
		return NewColumnNotExistsError(t.name, oldName)
	}

	pos := column.position

	// generate Position struct for creating new column
	if position == nil {
		position = &tidbast.ColumnPosition{Tp: tidbast.ColumnPositionNone}
	}
	if position.Tp == tidbast.ColumnPositionNone {
		if pos.value == 1 {
			position.Tp = tidbast.ColumnPositionFirst
		} else {
			for _, col := range t.columnSet {
				if col.position.value == pos.value-1 {
					position.Tp = tidbast.ColumnPositionAfter
					position.RelativeColumn = &tidbast.ColumnName{Name: model.NewCIStr(col.name)}
					break
				}
			}
		}
	}

	// drop column from columnSet
	for _, col := range t.columnSet {
		if col.position.value > pos.value {
			col.position.value--
		}
	}
	delete(t.columnSet, column.name)

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(newColumn, position)
}

// incompleteTableChangeColumn changes column definition.
// It does not maintain the position of the column.
func (t *TableState) incompleteTableChangeColumn(oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	delete(t.columnSet, oldName)

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(newColumn, position)
}

func (t *TableState) changeColumn(oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	if t.complete {
		return t.completeTableChangeColumn(oldName, newColumn, position)
	}
	return t.incompleteTableChangeColumn(oldName, newColumn, position)
}

func (t *TableState) dropIndex(indexName string) *WalkThroughError {
	if t.complete {
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

func (t *TableState) dropColumn(columnName string) *WalkThroughError {
	column, exists := t.columnSet[columnName]
	if !exists && t.complete {
		return NewColumnNotExistsError(t.name, columnName)
	}

	// Can not drop all columns in a table using ALTER TABLE DROP COLUMN.
	if t.complete && len(t.columnSet) == 1 {
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
		if index.expressionList.len() == 0 {
			delete(t.indexSet, index.name)
		}
	}

	// modify the column position
	if t.complete {
		for _, col := range t.columnSet {
			if col.position.value > column.position.value {
				col.position.value--
			}
		}
	}

	delete(t.columnSet, columnName)
	return nil
}

func (idx *IndexState) dropColumn(columnName string) {
	if !idx.expressionList.defined {
		return
	}
	var newKeyList []string
	for _, key := range idx.expressionList.value {
		if key != columnName {
			newKeyList = append(newKeyList, key)
		}
	}

	idx.expressionList = newStateStringSlice(newKeyList)
}

// reorderColumn reorders the columns for new column and returns the new column position.
func (t *TableState) reorderColumn(position *tidbast.ColumnPosition) (int64, *WalkThroughError) {
	switch position.Tp {
	case tidbast.ColumnPositionNone:
		return int64(len(t.columnSet) + 1), nil
	case tidbast.ColumnPositionFirst:
		for _, column := range t.columnSet {
			column.position.value++
		}
		return 1, nil
	case tidbast.ColumnPositionAfter:
		columnName := position.RelativeColumn.Name.O
		column, exist := t.columnSet[columnName]
		if !exist {
			return 0, NewColumnNotExistsError(t.name, columnName)
		}
		for _, col := range t.columnSet {
			if col.position.value > column.position.value {
				col.position.value++
			}
		}
		return column.position.value + 1, nil
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
			if name.Schema.O != "" && d.name != "" && d.name != name.Schema.O {
				return &WalkThroughError{
					Type:    ErrorTypeAccessOtherDatabase,
					Content: fmt.Sprintf("Database `%s` is not the current database `%s`", name.Schema.O, d.name),
				}
			}

			schema, exists := d.schemaSet[""]
			if !exists {
				schema = d.createSchema("")
			}

			if _, exists = schema.tableSet[name.Name.O]; !exists {
				if node.IfExists || !schema.complete {
					return nil
				}
				return &WalkThroughError{
					Type:    ErrorTypeTableNotExists,
					Content: fmt.Sprintf("Table `%s` does not exist", name.Name.O),
				}
			}

			delete(schema.tableSet, name.Name.O)
		}
	}
	return nil
}

func (d *DatabaseState) copyTable(node *tidbast.CreateTableStmt) *WalkThroughError {
	targetTable, err := d.findTableState(node.ReferTable)
	if err != nil {
		return err
	}
	if targetTable == nil {
		// For CREATE TABLE ... LIKE ... statements, we can not walk-through if target table dese not exist in catalog.
		return NewTableNotExistsError(node.ReferTable.Name.O)
	}

	schema := d.schemaSet[""]
	table := targetTable.copy()
	table.name = node.Table.Name.O
	schema.tableSet[table.name] = table
	return nil
}

func (d *DatabaseState) createTable(node *tidbast.CreateTableStmt) *WalkThroughError {
	if node.Table.Schema.O != "" && d.name != "" && d.name != node.Table.Schema.O {
		return &WalkThroughError{
			Type:    ErrorTypeAccessOtherDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", node.Table.Schema.O, d.name),
		}
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema("")
	}

	if _, exists = schema.tableSet[node.Table.Name.O]; exists {
		if node.IfNotExists {
			return nil
		}
		return &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf("Table `%s` already exists", node.Table.Name.O),
		}
	}

	if node.ReferTable != nil {
		return d.copyTable(node)
	}

	table := &TableState{
		complete:  true,
		name:      node.Table.Name.O,
		tableType: newStateString(""),
		engine:    newStateString(""),
		collation: newStateString(""),
		comment:   newStateString(""),
		columnSet: make(columnStateMap),
		indexSet:  make(indexStateMap),
	}
	schema.tableSet[table.name] = table

	for _, column := range node.Cols {
		if err := table.createColumn(column, nil /* position */); err != nil {
			err.Line = column.OriginTextPosition()
			return err
		}
	}

	for _, constraint := range node.Constraints {
		if err := table.createConstraint(constraint); err != nil {
			err.Line = constraint.OriginTextPosition()
			return err
		}
	}

	return nil
}

func (t *TableState) createConstraint(constraint *tidbast.Constraint) *WalkThroughError {
	switch constraint.Tp {
	case tidbast.ConstraintPrimaryKey:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, true /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createPrimaryKey(keyList, getIndexType(constraint.Option)); err != nil {
			return err
		}
	case tidbast.ConstraintKey, tidbast.ConstraintIndex:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, false /* unique */, getIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, false /* primary */, false /* isSpatial */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, true /* unique */, getIndexType(constraint.Option), constraint.Option); err != nil {
			return err
		}
	case tidbast.ConstraintForeignKey:
		// we do not deal with FOREIGN KEY constraints
	case tidbast.ConstraintFulltext:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, false /* primary */, false /* isSpatial */)
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

func (t *TableState) validateAndGetKeyStringList(keyList []*tidbast.IndexPartSpecification, primary bool, isSpatial bool) ([]string, *WalkThroughError) {
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
				if t.complete {
					return nil, NewColumnNotExistsError(t.name, columnName)
				}
			} else {
				if primary {
					column.nullable = newStateBool(false)
				}
				if isSpatial && column.nullable.defined && column.nullable.value {
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

func (t *TableState) createColumn(column *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	if _, exists := t.columnSet[column.Name.Name.O]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", column.Name.Name.O, t.name),
		}
	}

	pos := int64(len(t.columnSet) + 1)
	if position != nil && t.complete {
		var err *WalkThroughError
		pos, err = t.reorderColumn(position)
		if err != nil {
			return err
		}
	}

	col := &ColumnState{
		complete:     true,
		name:         column.Name.Name.O,
		position:     newStateInt(pos),
		defaultValue: newStateStringPointer(nil),
		nullable:     newStateBool(true),
		columnType:   newStateString(column.Tp.CompactStr()),
		characterSet: newStateString(column.Tp.GetCharset()),
		collation:    newStateString(column.Tp.GetCollate()),
		comment:      newStateString(""),
	}

	for _, option := range column.Options {
		switch option.Tp {
		case tidbast.ColumnOptionPrimaryKey:
			col.nullable = newStateBool(false)
			if err := t.createPrimaryKey([]string{col.name}, model.IndexTypeBtree.String()); err != nil {
				return err
			}
		case tidbast.ColumnOptionNotNull:
			col.nullable = newStateBool(false)
		case tidbast.ColumnOptionAutoIncrement:
			// we do not deal with AUTO-INCREMENT
		case tidbast.ColumnOptionDefaultValue:
			defaultValue, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.defaultValue = newStateStringPointer(&defaultValue)
		case tidbast.ColumnOptionUniqKey:
			if err := t.createIndex("", []string{col.name}, true /* unique */, model.IndexTypeBtree.String(), nil); err != nil {
				return err
			}
		case tidbast.ColumnOptionNull:
			col.nullable = newStateBool(true)
		case tidbast.ColumnOptionOnUpdate:
			// we do not deal with ON UPDATE
		case tidbast.ColumnOptionComment:
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.comment = newStateString(comment)
		case tidbast.ColumnOptionGenerated:
			// we do not deal with GENERATED ALWAYS AS
		case tidbast.ColumnOptionReference:
			// MySQL will ignore the inline REFERENCE
			// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		case tidbast.ColumnOptionCollate:
			col.collation = newStateString(option.StrValue)
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
		complete:       true,
		name:           name,
		expressionList: newStateStringSlice(keyList),
		indextype:      newStateString(tp),
		unique:         newStateBool(unique),
		primary:        newStateBool(false),
		visible:        newStateBool(true),
		comment:        newStateString(""),
	}

	if option != nil && option.Visibility == tidbast.IndexVisibilityInvisible {
		index.visible = newStateBool(false)
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
		expressionList: newStateStringSlice(keys),
		indextype:      newStateString(tp),
		unique:         newStateBool(true),
		primary:        newStateBool(true),
		visible:        newStateBool(true),
		comment:        newStateString(""),
	}
	t.indexSet[pk.name] = pk
	return nil
}

func (d *DatabaseState) createSchema(name string) *SchemaState {
	schema := &SchemaState{
		complete:     d.complete,
		name:         name,
		tableSet:     make(tableStateMap),
		viewSet:      make(viewStateMap),
		extensionSet: make(extensionStateMap),
	}

	d.schemaSet[name] = schema
	return schema
}

func (d *DatabaseState) parse(stmts string) ([]tidbast.StmtNode, *WalkThroughError) {
	p := tidbparser.New()
	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	nodeList, _, err := p.Parse(stmts, d.characterSet.value, d.collation.value)
	if err != nil {
		return nil, NewParseError(err.Error())
	}

	// sikp the setting line stage
	if len(nodeList) == 0 {
		return nodeList, nil
	}

	sqlList, err := parser.SplitMultiSQL(parser.MySQL, stmts)
	if err != nil {
		return nil, NewParseError(err.Error())
	}
	if len(sqlList) != len(nodeList) {
		return nil, NewParseError(fmt.Sprintf("split multi-SQL failed: the length should be %d, but get %d. stmt: \"%s\"", len(nodeList), len(sqlList), stmts))
	}

	for i, node := range nodeList {
		node.SetText(nil, strings.TrimSpace(node.Text()))
		node.SetOriginTextPosition(sqlList[i].LastLine)
		if n, ok := node.(*tidbast.CreateTableStmt); ok {
			if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, NewParseError(err.Error())
			}
		}
	}

	return nodeList, nil
}

func restoreNode(node tidbast.Node, flag format.RestoreFlags) (string, *WalkThroughError) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", &WalkThroughError{
			Type:    ErrorTypeRestoreError,
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

func getTableSourceName(table *tidbast.TableRefsClause) (*tidbast.TableName, bool) {
	source, isTableSource := table.TableRefs.Left.(*tidbast.TableSource)
	nilRight := table.TableRefs.Right == nil
	if isTableSource && nilRight {
		// isTableSource and nilRight mean it's a table.
		if tableName, ok := source.Source.(*tidbast.TableName); ok {
			return tableName, true
		}
	}
	return nil, false
}
