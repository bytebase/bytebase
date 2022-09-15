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
func (d *databaseState) WalkThrough(stmts string) error {
	if d.dbType != db.MySQL {
		return &WalkThroughError{
			Type:    ErrorTypeUnsupported,
			Content: fmt.Sprintf("Engine type %s is not supported", d.dbType),
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
		if err := d.changeState(node); err != nil {
			return err
		}
	}

	return nil
}

func (d *databaseState) changeState(in tidbast.StmtNode) error {
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

func (d *databaseState) renameTable(node *tidbast.RenameTableStmt) error {
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

func (d *databaseState) targetDatabase(node *tidbast.TableToTable) string {
	if node.OldTable.Schema.O != "" && node.OldTable.Schema.O != d.name {
		return node.OldTable.Schema.O
	}
	return node.NewTable.Schema.O
}

func (d *databaseState) moveToOtherDatabase(node *tidbast.TableToTable) bool {
	if node.OldTable.Schema.O != "" && node.OldTable.Schema.O != d.name {
		return false
	}
	return node.OldTable.Schema.O != node.NewTable.Schema.O
}

func (d *databaseState) theCurrentDatabase(node *tidbast.TableToTable) bool {
	if node.NewTable.Schema.O != "" && node.NewTable.Schema.O != d.name {
		return false
	}
	if node.OldTable.Schema.O != "" && node.OldTable.Schema.O != d.name {
		return false
	}
	return true
}

func (d *databaseState) dropDatabase(node *tidbast.DropDatabaseStmt) error {
	if node.Name != d.name {
		return NewAccessOtherDatabaseError(d.name, node.Name)
	}

	d.deleted = true
	return nil
}

func (d *databaseState) alterDatabase(node *tidbast.AlterDatabaseStmt) error {
	if !node.AlterDefaultDatabase && node.Name != d.name {
		return NewAccessOtherDatabaseError(d.name, node.Name)
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

func (d *databaseState) findTable(tableName *tidbast.TableName) (*tableState, error) {
	if tableName.Schema.O != "" && tableName.Schema.O != d.name {
		return nil, NewAccessOtherDatabaseError(d.name, tableName.Schema.O)
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema("")
	}

	table, exists := schema.tableSet[tableName.Name.O]
	if !exists {
		if !schema.context.CheckIntegrity {
			return nil, nil
		}

		return nil, NewTableNotExistsError(tableName.Name.O)
	}

	return table, nil
}

func (d *databaseState) dropIndex(node *tidbast.DropIndexStmt) error {
	table, err := d.findTable(node.Table)
	if err != nil {
		return err
	}
	if table == nil {
		return nil
	}

	return table.dropIndex(node.IndexName)
}

func (d *databaseState) createIndex(node *tidbast.CreateIndexStmt) error {
	table, err := d.findTable(node.Table)
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

func (d *databaseState) alterTable(node *tidbast.AlterTableStmt) error {
	table, err := d.findTable(node.Table)
	if err != nil {
		return err
	}
	if table == nil {
		return nil
	}

	for _, spec := range node.Specs {
		switch spec.Tp {
		case tidbast.AlterTableOption:
			for _, option := range spec.Options {
				switch option.Tp {
				case tidbast.TableOptionCollate:
					table.collation = option.StrValue
				case tidbast.TableOptionComment:
					table.comment = option.StrValue
				case tidbast.TableOptionEngine:
					table.engine = option.StrValue
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

func (t *tableState) changeIndexVisibility(indexName string, visibility tidbast.IndexVisibility) error {
	index, exists := t.indexSet[indexName]
	if !exists {
		return NewIndexNotExistsError(t.name, indexName)
	}
	switch visibility {
	case tidbast.IndexVisibilityVisible:
		index.visible = true
	case tidbast.IndexVisibilityInvisible:
		index.visible = false
	}
	return nil
}

func (t *tableState) renameIndex(oldName string, newName string) error {
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
		return NewIndexNotExistsError(t.name, oldName)
	}

	if _, exists := t.indexSet[newName]; exists {
		return NewIndexExistsError(t.name, newName)
	}

	index.name = newName
	delete(t.indexSet, oldName)
	t.indexSet[newName] = index
	return nil
}

func (t *tableState) changeColumnDefault(column *tidbast.ColumnDef) error {
	columnName := column.Name.Name.O
	colState, exists := t.columnSet[columnName]
	if !exists {
		return NewColumnNotExistsError(t.name, columnName)
	}

	if len(column.Options) == 1 {
		// SET DEFAULT
		defaultValue, err := restoreNode(column.Options[0].Expr, format.RestoreStringWithoutCharset)
		if err != nil {
			return err
		}
		colState.defaultValue = &defaultValue
	} else {
		// DROP DEFAULT
		colState.defaultValue = nil
	}
	return nil
}

func (s *schemaState) renameTable(oldName string, newName string) error {
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

func (t *tableState) renameColumn(oldName string, newName string) error {
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

func (t *tableState) renameColumnInIndexKey(oldName string, newName string) {
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

// changeColumn changes column definition.
// It works as:
// 1. drop column from tableState.columnSet, but do not drop column from indexSet.
// 2. rename column from indexSet.
// 3. create a new column in columnSet.
func (t *tableState) changeColumn(oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) error {
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
		if pos == 1 {
			position.Tp = tidbast.ColumnPositionFirst
		} else {
			for _, col := range t.columnSet {
				if col.position == pos-1 {
					position.Tp = tidbast.ColumnPositionAfter
					position.RelativeColumn = &tidbast.ColumnName{Name: model.NewCIStr(col.name)}
					break
				}
			}
		}
	}

	// drop column from columnSet
	for _, col := range t.columnSet {
		if col.position > pos {
			col.position--
		}
	}
	delete(t.columnSet, column.name)

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(newColumn, position)
}

func (t *tableState) dropIndex(indexName string) error {
	if _, exists := t.indexSet[indexName]; !exists {
		if indexName == PrimaryKeyName {
			return &WalkThroughError{
				Type:    ErrorTypePrimaryKeyNotExists,
				Content: fmt.Sprintf("Primary key does not exist in table `%s`", t.name),
			}
		}
		return NewIndexNotExistsError(t.name, indexName)
	}

	delete(t.indexSet, indexName)
	return nil
}

func (t *tableState) dropColumn(columnName string) error {
	column, exists := t.columnSet[columnName]
	if !exists {
		return NewColumnNotExistsError(t.name, columnName)
	}

	// Can not drop all columns in a table using ALTER TABLE DROP COLUMN.
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
		if col.position > column.position {
			col.position--
		}
	}

	delete(t.columnSet, columnName)
	return nil
}

func (idx *indexState) dropColumn(columnName string) {
	var newKeyList []string
	for _, key := range idx.expressionList {
		if key != columnName {
			newKeyList = append(newKeyList, key)
		}
	}

	idx.expressionList = newKeyList
}

// reorderColumn reorders the columns for new column and returns the new column position.
func (t *tableState) reorderColumn(position *tidbast.ColumnPosition) (int, error) {
	switch position.Tp {
	case tidbast.ColumnPositionNone:
		return len(t.columnSet) + 1, nil
	case tidbast.ColumnPositionFirst:
		for _, column := range t.columnSet {
			column.position++
		}
		return 1, nil
	case tidbast.ColumnPositionAfter:
		columnName := position.RelativeColumn.Name.O
		column, exist := t.columnSet[columnName]
		if !exist {
			return 0, NewColumnNotExistsError(t.name, columnName)
		}
		for _, col := range t.columnSet {
			if col.position > column.position {
				col.position++
			}
		}
		return column.position + 1, nil
	}
	return 0, &WalkThroughError{
		Type:    ErrorTypeUnsupported,
		Content: fmt.Sprintf("Unsupported column position type: %d", position.Tp),
	}
}

func (d *databaseState) dropTable(node *tidbast.DropTableStmt) error {
	// TODO(rebelice): deal with DROP VIEW statement.
	if !node.IsView {
		for _, name := range node.Tables {
			if name.Schema.O != "" && d.name != name.Schema.O {
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
				if node.IfExists || !schema.context.CheckIntegrity {
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

func (d *databaseState) copyTable(node *tidbast.CreateTableStmt) error {
	targetTable, err := d.findTable(node.ReferTable)
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

func (d *databaseState) createTable(node *tidbast.CreateTableStmt) error {
	if node.Table.Schema.O != "" && d.name != node.Table.Schema.O {
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

	table := &tableState{
		name:      node.Table.Name.O,
		columnSet: make(columnStateMap),
		indexSet:  make(indexStateMap),
	}
	schema.tableSet[table.name] = table

	for _, column := range node.Cols {
		if err := table.createColumn(column, nil); err != nil {
			return err
		}
	}

	for _, constraint := range node.Constraints {
		if err := table.createConstraint(constraint); err != nil {
			return err
		}
	}

	return nil
}

func (t *tableState) createConstraint(constraint *tidbast.Constraint) error {
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

func (t *tableState) validateAndGetKeyStringList(keyList []*tidbast.IndexPartSpecification, primary bool, isSpatial bool) ([]string, error) {
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
				return nil, NewColumnNotExistsError(t.name, columnName)
			}
			if primary {
				column.nullable = false
			}
			if isSpatial && column.nullable {
				return nil, &WalkThroughError{
					Type: ErrorTypeSpatialIndexKeyNullable,
					// The error content comes from MySQL.
					Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name),
				}
			}
			res = append(res, columnName)
		}
	}
	return res, nil
}

func (t *tableState) createColumn(column *tidbast.ColumnDef, position *tidbast.ColumnPosition) error {
	if _, exists := t.columnSet[column.Name.Name.O]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", column.Name.Name.O, t.name),
		}
	}

	pos := len(t.columnSet) + 1
	if position != nil {
		var err error
		pos, err = t.reorderColumn(position)
		if err != nil {
			return err
		}
	}

	col := &columnState{
		name:         column.Name.Name.O,
		position:     pos,
		columnType:   column.Tp.CompactStr(),
		characterSet: column.Tp.GetCharset(),
		collation:    column.Tp.GetCollate(),
		nullable:     true,
	}

	for _, option := range column.Options {
		switch option.Tp {
		case tidbast.ColumnOptionPrimaryKey:
			col.nullable = false
			if err := t.createPrimaryKey([]string{col.name}, model.IndexTypeBtree.String()); err != nil {
				return err
			}
		case tidbast.ColumnOptionNotNull:
			col.nullable = false
		case tidbast.ColumnOptionAutoIncrement:
			// we do not deal with AUTO-INCREMENT
		case tidbast.ColumnOptionDefaultValue:
			defaultValue, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.defaultValue = &defaultValue
		case tidbast.ColumnOptionUniqKey:
			if err := t.createIndex("", []string{col.name}, true /* unique */, model.IndexTypeBtree.String(), nil); err != nil {
				return err
			}
		case tidbast.ColumnOptionNull:
			col.nullable = true
		case tidbast.ColumnOptionOnUpdate:
			// we do not deal with ON UPDATE
		case tidbast.ColumnOptionComment:
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.comment = comment
		case tidbast.ColumnOptionGenerated:
			// we do not deal with GENERATED ALWAYS AS
		case tidbast.ColumnOptionReference:
			// MySQL will ignore the inline REFERENCE
			// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		case tidbast.ColumnOptionCollate:
			col.collation = option.StrValue
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

func (t *tableState) createIndex(name string, keyList []string, unique bool, tp string, option *tidbast.IndexOption) error {
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

	index := &indexState{
		name:           name,
		expressionList: keyList,
		unique:         unique,
		primary:        false,
		indextype:      tp,
		visible:        true,
	}

	if option != nil && option.Visibility == tidbast.IndexVisibilityInvisible {
		index.visible = false
	}

	t.indexSet[name] = index
	return nil
}

func (t *tableState) createPrimaryKey(keys []string, tp string) error {
	if _, exists := t.indexSet[PrimaryKeyName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypePrimaryKeyExists,
			Content: fmt.Sprintf("Primary key exists in table `%s`", t.name),
		}
	}

	pk := &indexState{
		name:           PrimaryKeyName,
		expressionList: keys,
		unique:         true,
		primary:        true,
		indextype:      tp,
		visible:        true,
	}
	t.indexSet[pk.name] = pk
	return nil
}

func (d *databaseState) createSchema(name string) *schemaState {
	schema := &schemaState{
		name:         name,
		tableSet:     make(tableStateMap),
		viewSet:      make(viewStateMap),
		extensionSet: make(extensionStateMap),
		context:      d.context.Copy(),
	}

	d.schemaSet[name] = schema
	return schema
}

func (d *databaseState) parse(stmts string) ([]tidbast.StmtNode, error) {
	p := tidbparser.New()
	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	nodeList, _, err := p.Parse(stmts, d.characterSet, d.collation)
	if err != nil {
		return nil, NewParseError(err.Error())
	}
	sqlList, err := parser.SplitMultiSQL(parser.MySQL, stmts)
	if err != nil {
		return nil, NewParseError(err.Error())
	}
	if len(sqlList) != len(nodeList) {
		return nil, NewParseError(fmt.Sprintf("split multi-SQL failed: the length should be %d, but get %d. stmt: \"%s\"", len(nodeList), len(sqlList), stmts))
	}

	for i, node := range nodeList {
		node.SetOriginTextPosition(sqlList[i].Line)
		if n, ok := node.(*tidbast.CreateTableStmt); ok {
			if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, NewParseError(err.Error())
			}
		}
	}

	return nodeList, nil
}

func restoreNode(node tidbast.Node, flag format.RestoreFlags) (string, error) {
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
