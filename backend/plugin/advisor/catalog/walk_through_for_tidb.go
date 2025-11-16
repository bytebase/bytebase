package catalog

import (
	"fmt"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

// TiDBWalkThrough walks through TiDB AST and updates the database state.
func TiDBWalkThrough(d *DatabaseState, ast any) error {
	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	d.GetOrCreateSchema("")

	nodeList, ok := ast.([]tidbast.StmtNode)
	if !ok {
		return errors.Errorf("invalid ast type %T", ast)
	}
	for _, node := range nodeList {
		// change state
		if err := d.changeState(node); err != nil {
			return err
		}
	}

	return nil
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
	if d.IsDeleted() {
		return &WalkThroughError{
			Code:    code.DatabaseIsDeleted,
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
			schema = d.createSchema()
		}
		oldTableName := tableToTable.OldTable.Name.O
		newTableName := tableToTable.NewTable.Name.O
		if d.theCurrentDatabase(tableToTable) {
			if compareIdentifier(oldTableName, newTableName, d.ignoreCaseSensitive) {
				return nil
			}
			table, exists := schema.getTable(oldTableName)
			if !exists {
				return NewTableNotExistsError(oldTableName)
			}
			if _, exists := schema.getTable(newTableName); exists {
				return NewTableExistsError(newTableName)
			}
			delete(schema.tableSet, table.name)
			table.name = newTableName
			schema.tableSet[table.name] = table
		} else if d.moveToOtherDatabase(tableToTable) {
			_, exists := schema.getTable(tableToTable.OldTable.Name.O)
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

	d.MarkDeleted()
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
		default:
			// Other database options
		}
	}
	return nil
}

func (d *DatabaseState) findTableState(tableName *tidbast.TableName) (*TableState, *WalkThroughError) {
	if tableName.Schema.O != "" && !d.isCurrentDatabase(tableName.Schema.O) {
		return nil, NewAccessOtherDatabaseError(d.name, tableName.Schema.O)
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema()
	}

	table, exists := schema.getTable(tableName.Name.O)
	if !exists {
		return nil, NewTableNotExistsError(tableName.Name.O)
	}

	return table, nil
}

func (d *DatabaseState) dropIndex(node *tidbast.DropIndexStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}

	return table.DropIndex(node.IndexName, nil)
}

func (d *DatabaseState) createIndex(node *tidbast.CreateIndexStmt) *WalkThroughError {
	table, err := d.findTableState(node.Table)
	if err != nil {
		return err
	}

	unique := false
	tp := tidbast.IndexTypeBtree.String()
	isSpatial := false

	switch node.KeyType {
	case tidbast.IndexKeyTypeNone:
	case tidbast.IndexKeyTypeUnique:
		unique = true
	case tidbast.IndexKeyTypeFulltext:
		tp = FullTextName
	case tidbast.IndexKeyTypeSpatial:
		isSpatial = true
		tp = SpatialName
	default:
		// Other index key types
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
					table.collation = newStringPointer(option.StrValue)
				case tidbast.TableOptionComment:
					table.comment = newStringPointer(option.StrValue)
				case tidbast.TableOptionEngine:
					table.engine = newStringPointer(option.StrValue)
				default:
					// Other table options
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
			if err := table.DropColumn(spec.OldColumnName.Name.O, nil); err != nil {
				return err
			}
		case tidbast.AlterTableDropPrimaryKey:
			if err := table.DropIndex(PrimaryKeyName, nil); err != nil {
				return err
			}
		case tidbast.AlterTableDropIndex:
			if err := table.DropIndex(spec.Name, nil); err != nil {
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
			if err := table.RenameColumn(spec.OldColumnName.Name.O, spec.NewColumnName.Name.O); err != nil {
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
			if err := table.RenameIndex(spec.FromKey.O, spec.ToKey.O, nil); err != nil {
				return err
			}
		case tidbast.AlterTableIndexInvisible:
			if err := table.changeIndexVisibility(spec.IndexName.O, spec.Visibility); err != nil {
				return err
			}
		default:
			// Other ALTER TABLE types
		}
	}

	return nil
}

func (t *TableState) changeIndexVisibility(indexName string, visibility tidbast.IndexVisibility) *WalkThroughError {
	index, exists := t.indexSet[strings.ToLower(indexName)]
	if !exists {
		return NewIndexNotExistsError(t.name, indexName)
	}
	switch visibility {
	case tidbast.IndexVisibilityVisible:
		index.visible = newTruePointer()
	case tidbast.IndexVisibilityInvisible:
		index.visible = newFalsePointer()
	default:
		// Keep current visibility
	}
	return nil
}

func (t *TableState) changeColumnDefault(column *tidbast.ColumnDef) *WalkThroughError {
	columnName := column.Name.Name.L
	colState, exists := t.columnSet[columnName]
	if !exists {
		return NewColumnNotExistsError(t.name, columnName)
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
						Code: code.InvalidColumnDefault,
						// Content comes from MySQL Error content.
						Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
					}
				default:
					// Other column types allow default values
				}
			}

			defaultValue, err := restoreNode(column.Options[0].Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return &WalkThroughError{
					Code:    code.Internal,
					Content: fmt.Sprintf("Failed to deparse default value: %v", err),
				}
			}
			colState.defaultValue = &defaultValue
		} else {
			if colState.nullable != nil && !*colState.nullable {
				return &WalkThroughError{
					Code: code.SetNullDefaultForNotNullColumn,
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

func (t *TableState) renameColumnInIndexKey(oldName string, newName string) {
	if strings.EqualFold(oldName, newName) {
		return
	}
	for _, index := range t.indexSet {
		for i, key := range index.expressionList {
			if strings.EqualFold(key, oldName) {
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
func (t *TableState) completeTableChangeColumn(oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	column, exists := t.columnSet[strings.ToLower(oldName)]
	if !exists {
		return NewColumnNotExistsError(t.name, oldName)
	}

	pos := *column.position

	// generate Position struct for creating new column
	// Create a local copy to avoid modifying the input parameter
	var localPosition *tidbast.ColumnPosition
	if position == nil {
		localPosition = &tidbast.ColumnPosition{Tp: tidbast.ColumnPositionNone}
	} else {
		// Create a copy of the position to avoid modifying the original
		localPosition = &tidbast.ColumnPosition{
			Tp:             position.Tp,
			RelativeColumn: position.RelativeColumn,
		}
	}

	if localPosition.Tp == tidbast.ColumnPositionNone {
		if pos == 1 {
			localPosition.Tp = tidbast.ColumnPositionFirst
		} else {
			for _, col := range t.columnSet {
				if *col.position == pos-1 {
					localPosition.Tp = tidbast.ColumnPositionAfter
					localPosition.RelativeColumn = &tidbast.ColumnName{Name: tidbast.NewCIStr(col.name)}
					break
				}
			}
		}
	}
	position = localPosition

	// drop column from columnSet
	for _, col := range t.columnSet {
		if *col.position > pos {
			*col.position--
		}
	}
	delete(t.columnSet, strings.ToLower(column.name))

	// rename column from indexSet
	t.renameColumnInIndexKey(oldName, newColumn.Name.Name.O)

	// create a new column in columnSet
	return t.createColumn(newColumn, position)
}

func (t *TableState) changeColumn(oldName string, newColumn *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	return t.completeTableChangeColumn(oldName, newColumn, position)
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
		columnName := position.RelativeColumn.Name.L
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
	default:
		return 0, &WalkThroughError{
			Code:    code.Unsupported,
			Content: fmt.Sprintf("Unsupported column position type: %d", position.Tp),
		}
	}
}

func (d *DatabaseState) dropTable(node *tidbast.DropTableStmt) *WalkThroughError {
	// TODO(rebelice): deal with DROP VIEW statement.
	if !node.IsView {
		for _, name := range node.Tables {
			if name.Schema.O != "" && !d.isCurrentDatabase(name.Schema.O) {
				return &WalkThroughError{
					Code:    code.NotCurrentDatabase,
					Content: fmt.Sprintf("Database `%s` is not the current database `%s`", name.Schema.O, d.name),
				}
			}

			schema, exists := d.schemaSet[""]
			if !exists {
				schema = d.createSchema()
			}

			table, exists := schema.getTable(name.Name.O)
			if !exists {
				if node.IfExists {
					return nil
				}
				return &WalkThroughError{
					Code:    code.TableNotExists,
					Content: fmt.Sprintf("Table `%s` does not exist", name.Name.O),
				}
			}

			delete(schema.tableSet, table.name)
		}
	}
	return nil
}

func (d *DatabaseState) copyTable(node *tidbast.CreateTableStmt) *WalkThroughError {
	targetTable, err := d.findTableState(node.ReferTable)
	if err != nil {
		if err.Code == code.NotCurrentDatabase {
			return &WalkThroughError{
				Code:    code.ReferenceOtherDatabase,
				Content: fmt.Sprintf("Reference table `%s` in other database `%s`, skip walkthrough", node.ReferTable.Name.O, node.ReferTable.Schema.O),
			}
		}
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
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", node.Table.Schema.O, d.name),
		}
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema()
	}

	if _, exists = schema.getTable(node.Table.Name.O); exists {
		if node.IfNotExists {
			return nil
		}
		return &WalkThroughError{
			Code:    code.TableExists,
			Content: fmt.Sprintf("Table `%s` already exists", node.Table.Name.O),
		}
	}

	if node.Select != nil {
		return &WalkThroughError{
			Code:    code.StatementCreateTableAs,
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
					Code: code.AutoIncrementExists,
					// The content comes from MySQL error content.
					Content: fmt.Sprintf("There can be only one auto column for table `%s`", table.name),
				}
			}
			hasAutoIncrement = true
		}
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
	default:
		// Other constraint types
	}

	return nil
}

func (t *TableState) validateAndGetKeyStringList(keyList []*tidbast.IndexPartSpecification, primary bool, isSpatial bool) ([]string, *WalkThroughError) {
	var res []string
	for _, key := range keyList {
		if key.Expr != nil {
			str, err := restoreNode(key, format.DefaultRestoreFlags)
			if err != nil {
				return nil, &WalkThroughError{
					Code:    code.Internal,
					Content: fmt.Sprintf("Failed to deparse index key: %v", err),
				}
			}
			res = append(res, str)
		} else {
			columnName := key.Column.Name.L
			column, exists := t.columnSet[columnName]
			if !exists {
				return nil, NewColumnNotExistsError(t.name, columnName)
			}
			if primary {
				column.nullable = newFalsePointer()
			}
			if isSpatial && column.nullable != nil && *column.nullable {
				return nil, &WalkThroughError{
					Code: code.SpatialIndexKeyNullable,
					// The error content comes from MySQL.
					Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name),
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
				Code: code.InvalidColumnDefault,
				// Content comes from MySQL Error content.
				Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
			}
		default:
			// Other column types allow default values
		}
	}

	if valueExpr, yes := value.(tidbast.ValueExpr); yes {
		datum := types.NewDatum(valueExpr.GetValue())
		if _, err := datum.ConvertTo(types.Context{}, columnType); err != nil {
			return &WalkThroughError{
				Code:    code.InvalidColumnDefault,
				Content: err.Error(),
			}
		}
	}
	return nil
}

//nolint:revive
func (t *TableState) createColumn(column *tidbast.ColumnDef, position *tidbast.ColumnPosition) *WalkThroughError {
	if _, exists := t.columnSet[column.Name.Name.L]; exists {
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", column.Name.Name.O, t.name),
		}
	}

	pos := len(t.columnSet) + 1
	if position != nil {
		var err *WalkThroughError
		pos, err = t.reorderColumn(position)
		if err != nil {
			return err
		}
	}

	vTrue := true
	col := &ColumnState{
		name:         column.Name.Name.L,
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
			if err := t.createPrimaryKey([]string{col.name}, tidbast.IndexTypeBtree.String()); err != nil {
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
					return &WalkThroughError{
						Code:    code.Internal,
						Content: fmt.Sprintf("Failed to deparse default value: %v", err),
					}
				}
				col.defaultValue = &defaultValue
			} else {
				setNullDefault = true
			}
		case tidbast.ColumnOptionUniqKey:
			if err := t.createIndex("", []string{col.name}, true /* unique */, tidbast.IndexTypeBtree.String(), nil); err != nil {
				return err
			}
		case tidbast.ColumnOptionNull:
			col.nullable = newTruePointer()
		case tidbast.ColumnOptionOnUpdate:
			// we do not deal with ON UPDATE
			if column.Tp.GetType() != mysql.TypeDatetime && column.Tp.GetType() != mysql.TypeTimestamp {
				return &WalkThroughError{
					Code:    code.OnUpdateColumnNotDatetimeOrTimestamp,
					Content: fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.name),
				}
			}
		case tidbast.ColumnOptionComment:
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return &WalkThroughError{
					Code:    code.Internal,
					Content: fmt.Sprintf("Failed to deparse comment: %v", err),
				}
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
		default:
			// Other column options
		}
	}

	if col.nullable != nil && !*col.nullable && setNullDefault {
		return &WalkThroughError{
			Code: code.SetNullDefaultForNotNullColumn,
			// Content comes from MySQL Error content.
			Content: fmt.Sprintf("Invalid default value for column `%s`", col.name),
		}
	}

	t.columnSet[strings.ToLower(col.name)] = col
	return nil
}

//nolint:revive
func (t *TableState) createIndex(name string, keyList []string, unique bool, tp string, option *tidbast.IndexOption) *WalkThroughError {
	if len(keyList) == 0 {
		return &WalkThroughError{
			Code:    code.IndexEmptyKeys,
			Content: fmt.Sprintf("Index `%s` in table `%s` has empty key", name, t.name),
		}
	}
	if name != "" {
		if _, exists := t.indexSet[strings.ToLower(name)]; exists {
			return NewIndexExistsError(t.name, name)
		}
	} else {
		suffix := 1
		for {
			name = keyList[0]
			if suffix > 1 {
				name = fmt.Sprintf("%s_%d", keyList[0], suffix)
			}
			if _, exists := t.indexSet[strings.ToLower(name)]; !exists {
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

	t.indexSet[strings.ToLower(name)] = index
	return nil
}

//nolint:revive
func (t *TableState) createPrimaryKey(keys []string, tp string) *WalkThroughError {
	if _, exists := t.indexSet[strings.ToLower(PrimaryKeyName)]; exists {
		return &WalkThroughError{
			Code:    code.PrimaryKeyExists,
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
	t.indexSet[strings.ToLower(pk.name)] = pk
	return nil
}

func restoreNode(node tidbast.Node, flag format.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", errors.Wrapf(err, "failed to deparse node")
	}
	return buffer.String(), nil
}

func getIndexType(option *tidbast.IndexOption) string {
	if option != nil {
		switch option.Tp {
		case tidbast.IndexTypeBtree,
			tidbast.IndexTypeHash,
			tidbast.IndexTypeRtree:
			return option.Tp.String()
		default:
			// Other index types
		}
	}
	return tidbast.IndexTypeBtree.String()
}
