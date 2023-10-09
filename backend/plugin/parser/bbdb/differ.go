package bbdb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_MYSQL, SchemaDiff)
}

// SchemaDiffer is the parser for MySQL dialect.
type SchemaDiffer struct {
}

// SchemaDiff returns the schema diff.
// It only supports schema information from mysqldump.
func SchemaDiff(oldStmt, newStmt string, ignoreCaseSensitive bool) (string, error) {
	// 1. Preprocessing Stage.
	diff := &diffNode{
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
	if err := diff.diffStatement(oldStmt, newStmt); err != nil {
		return "", err
	}

	return diff.deparse()
}

// diffNode defines different modification types as the safe change order.
// The safe change order means we can change them with no dependency conflicts as this order.
type diffNode struct {
	// Ignore the case sensitive when comparing the table and view names.
	ignoreCaseSensitive bool

	dropForeignKeyList      []*foreignKeyDef
	dropPrimaryKeyList      []*primaryKeyDef
	createPrimaryKeyList    []*primaryKeyDef
	dropIndexList           []*indexDef
	dropViewList            []*viewDef
	dropTableList           []*tableDef
	dropCheckConstraintList []*checkDef
	addCheckConstraintList  []*checkDef

	createTableList      []*tableDef
	alterTableOptionList []*tableOptionDef
	addColumnList        []*columnDef
	modifyColumnList     []*columnDef
	dropColumnList       []*columnDef
	createTempViewList   []*viewDef
	createIndexList      []*indexDef
	addForeignKeyList    []*foreignKeyDef
	createViewList       []*viewDef

	dropEventList       []*eventDef
	createEventList     []*eventDef
	dropTriggerList     []*triggerDef
	createTriggerList   []*triggerDef
	dropFunctionList    []*functionDef
	createFunctionList  []*functionDef
	dropProcedureList   []*procedureDef
	createProcedureList []*procedureDef
}

func (diff *diffNode) diffStatement(oldStatement string, newStatement string) error {
	oldDatabaseDef, err := diff.buildSchemaInfo(oldStatement)
	if err != nil {
		return errors.Wrapf(err, "failed to parse old statement %q", oldStatement)
	}

	newDatabaseDef, err := diff.buildSchemaInfo(newStatement)
	if err != nil {
		return errors.Wrapf(err, "failed to parse new statement %q", newStatement)
	}

	if err := diff.diffTables(oldDatabaseDef, newDatabaseDef); err != nil {
		return errors.Wrapf(err, "failed to diff view")
	}

	if err := diff.diffView(oldDatabaseDef, newDatabaseDef); err != nil {
		return errors.Wrapf(err, "failed to diff view")
	}

	if err := diff.diffFunction(oldDatabaseDef, newDatabaseDef); err != nil {
		return errors.Wrapf(err, "failed to diff function")
	}

	if err := diff.diffProcedure(oldDatabaseDef, newDatabaseDef); err != nil {
		return errors.Wrapf(err, "failed to diff procedure")
	}

	if err := diff.diffEvent(oldDatabaseDef.schemas[""], newDatabaseDef.schemas[""]); err != nil {
		return errors.Wrapf(err, "failed to diff event")
	}

	if err := diff.diffTrigger(oldDatabaseDef.schemas[""], newDatabaseDef.schemas[""]); err != nil {
		return errors.Wrapf(err, "failed to diff trigger")
	}

	return nil
}

func (diff *diffNode) diffTables(oldDatabase, newDatabase *databaseDef) error {
	for newTableName, newTable := range newDatabase.schemas[""].tables {
		oldTable, exists := oldDatabase.schemas[""].tables[newTableName]
		if !exists {
			diff.createTableList = append(diff.createTableList, newTable)
			// Create indexes.
			for _, index := range newTable.indexes {
				diff.createIndexList = append(diff.createIndexList, index)
			}
			continue
		}
		diff.diffTable(oldTable, newTable)
		delete(oldDatabase.schemas[""].tables, newTableName)
	}

	for _, oldTable := range oldDatabase.schemas[""].tables {
		diff.dropTableList = append(diff.dropTableList, oldTable)
	}

	return nil
}

func (diff *diffNode) diffTable(oldTable, newTable *tableDef) {
	diff.diffColumn(oldTable, newTable)
	diff.diffIndex(oldTable, newTable)
	diff.diffPrimaryKey(oldTable, newTable)
	diff.diffForeignKey(oldTable, newTable)
	diff.diffCheckConstraint(oldTable, newTable)
	diff.diffTableOptions(oldTable, newTable)
}

func (diff *diffNode) diffCheckConstraint(oldTable, newTable *tableDef) {
	for constraintName, constraint := range newTable.checks {
		if oldConstraint, ok := oldTable.checks[strings.ToLower(constraintName)]; ok {
			if !isCheckConstraintEqual(oldConstraint, constraint) {
				diff.dropCheckConstraintList = append(diff.dropCheckConstraintList, oldConstraint)
				diff.addCheckConstraintList = append(diff.addCheckConstraintList, constraint)
			}
			delete(oldTable.checks, strings.ToLower(constraintName))
			continue
		}
		diff.addCheckConstraintList = append(diff.addCheckConstraintList, constraint)
	}
	for _, constraint := range oldTable.checks {
		diff.dropCheckConstraintList = append(diff.dropCheckConstraintList, constraint)
	}
}

func isCheckConstraintEqual(old, new *checkDef) bool {
	if !strings.EqualFold(old.name, new.name) {
		return false
	}

	if old.ctx.GetText() != new.ctx.GetText() {
		return false
	}
	return true
}

func (diff *diffNode) diffTableOptions(oldTable, newTable *tableDef) {
	for oldTp, oldOption := range oldTable.tableOptions {
		newOption, ok := newTable.tableOptions[oldTp]
		if !ok {
			switch oldTp {
			// For table engine, table charset and table collation, if oldTable has but newTable doesn't,
			// we skip drop them.
			case "ENGINE", "CHARSET", "COLLATE":
				continue
			}
			// We should drop the table option if it doesn't exist in the new table options.
			if astOption := dropTableOption(oldOption); astOption != nil {
				diff.alterTableOptionList = append(diff.alterTableOptionList, astOption)
			}
			continue
		}
		if !isTableOptionEqual(oldOption, newOption) {
			diff.alterTableOptionList = append(diff.alterTableOptionList, newOption)
		}
	}

	for newTp, newOption := range newTable.tableOptions {
		if _, ok := oldTable.tableOptions[newTp]; !ok {
			diff.alterTableOptionList = append(diff.alterTableOptionList, newOption)
		}
	}
}

func dropTableOption(oldOption *tableOptionDef) *tableOptionDef {
	switch oldOption.option {
	case "ENGINE":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` ENGINE = InnoDB;", oldOption.tableName)
	case "SECONDARY_ENGINE_ATTRIBUTE":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` SECONDARY_ENGINE_ATTRIBUTE = InnoDB;", oldOption.tableName)
	case "DEFAULT CHARSET":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` CHARACTER SET = utf8mb4;", oldOption.tableName)
	case "DEFAULT COLLATE":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` DEFAULT COLLATE = utf8mb4_general_ci;", oldOption.tableName)
	case "AUTO_INCREMENT":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` AUTO_INCREMENT = 0;", oldOption.tableName)
	case "COMMENT":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` COMMENT = 0;", oldOption.tableName)
	case "AVG_ROW_LENGTH":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` AVG_ROW_LENGTH = 0;", oldOption.tableName)
	case "CHECKSUM":
	case "COMPRESSION":
	case "CONNECTION":
	case "PASSWORD":
	case "KEY_BLOCK_SIZE":
	case "MAX_ROWS":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` MAX_ROWS = 0;", oldOption.tableName)
	case "MIN_ROWS":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` MIN_ROWS = 0;", oldOption.tableName)
	case "DELAY_KEY_WRITE":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` DELAY_KEY_WRITE = 0;", oldOption.tableName)
	case "ROW_FORMAT":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` COMMENT = '';", oldOption.tableName)
	case "STATS_AUTO_RECALC":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` STATS_AUTO_RECALC = \"DEFAULT\";", oldOption.tableName)
	case "PACK_KEYS":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` PACK_KEYS = \"DEFAULT\";", oldOption.tableName)
	case "TABLESPACE":
	case "INSERT_METHOD":
	case "TABLE_CHECKSUM":
	case "UNION":
	case "ENCRYPTION":
	}
	return oldOption
}

// isTableOptionValEqual compare the two table options value, if they are equal, returns true.
// Caller need to ensure the two table options are not nil and the type is the same.
func isTableOptionEqual(oldOption, newOption *tableOptionDef) bool {
	return oldOption.ctx.GetText() == newOption.ctx.GetText()
}

func (diff *diffNode) diffPrimaryKey(oldTable, newTable *tableDef) {
	oldPrimaryKey := oldTable.primaryKey
	newPrimaryKey := newTable.primaryKey
	if oldPrimaryKey == nil && newPrimaryKey == nil {
		return
	} else if oldPrimaryKey != nil && newPrimaryKey == nil {
		diff.dropPrimaryKeyList = append(diff.dropPrimaryKeyList, oldPrimaryKey)
	} else if oldPrimaryKey == nil && newPrimaryKey != nil {
		diff.createPrimaryKeyList = append(diff.createPrimaryKeyList, newPrimaryKey)
	}
	if !isPrimaryKeyEqual(oldPrimaryKey, newPrimaryKey) {
		diff.dropPrimaryKeyList = append(diff.dropPrimaryKeyList, oldPrimaryKey)
		diff.createPrimaryKeyList = append(diff.createPrimaryKeyList, newPrimaryKey)
	}
}

// isPrimaryKeyEqual returns true if definitions of two priamry indexes are the same.
func isPrimaryKeyEqual(old, new *primaryKeyDef) bool {
	if old.tableName != new.tableName {
		return false
	}
	if !isKeyPartEqual(old.columns, new.columns) {
		return false
	}
	if old.ctx.GetText() != new.ctx.GetText() {
		return false
	}
	return true
}

func (diff *diffNode) diffForeignKey(oldTable, newTable *tableDef) {
	for _, foreignKey := range newTable.foreignKeys {
		if oldForeignKey, ok := oldTable.foreignKeys[strings.ToLower(foreignKey.name)]; ok {
			if !isForeignKeyEqual(oldForeignKey, foreignKey) {
				diff.dropForeignKeyList = append(diff.dropForeignKeyList, oldForeignKey)
				diff.addForeignKeyList = append(diff.addForeignKeyList, foreignKey)
			}
			delete(oldTable.foreignKeys, strings.ToLower(foreignKey.name))
			continue
		}
		diff.addForeignKeyList = append(diff.addForeignKeyList, foreignKey)
	}
}

// isForeignKeyEqual returns true if two foreign keys are the same.
func isForeignKeyEqual(old, new *foreignKeyDef) bool {
	if old.name != new.name {
		return false
	}
	if !isKeyPartEqual(old.columns, new.columns) {
		return false
	}
	if !isKeyPartEqual(old.referencedColumns, new.referencedColumns) {
		return false
	}
	if old.referencedTable != new.referencedTable {
		return false
	}
	return true
}

func (diff *diffNode) diffView(oldDatabase, newDatabase *databaseDef) error {
	var tempViewList []*viewDef
	for _, view := range newDatabase.views {
		viewName := view.name
		if diff.ignoreCaseSensitive {
			viewName = strings.ToLower(viewName)
		}

		oldView, ok := oldDatabase.views[viewName]
		if ok {
			if !diff.isViewEqual(view, oldView) {
				diff.createViewList = append(diff.createViewList, view)
			}
			// We should delete the view in the oldViewMap, because we will drop the all views in the oldViewMap explicitly at last.
			delete(oldDatabase.views, viewName)
		} else {
			// We should create the view.
			// We create the temporary view first and replace it to avoid break the dependency like mysqldump does.
			tempView, err := getTempView(view)
			if err != nil {
				return errors.Wrapf(err, "failed to get temporary view for view %s", view.name)
			}
			tempViewList = append(tempViewList, tempView)
			diff.createViewList = append(diff.createViewList, view)
		}
	}
	diff.createTempViewList = append(diff.createTempViewList, tempViewList...)
	for _, view := range oldDatabase.views {
		diff.dropViewList = append(diff.dropViewList, view)
	}
	return nil
}

// getTempView returns the temporary view name and the create statement.
func getTempView(view *viewDef) (*viewDef, error) {
	// We create the temp view similar to what mysqldump does.
	// Create a temporary view with the same name as the view. Its columns should
	// have the same name in order to satisfy views that depend on this view.
	// This temporary view will be removed when the actual view is created.
	// The column properties are unnecessary and not preserved in this temporary view.
	// because other views only need to reference the column name.
	definer := "current_user"
	if view.ctx.DefinerClause() != nil {
		definer = view.ctx.DefinerClause().GetText()
	}
	sqlSecurity := "DEFINER"
	if view.ctx.ViewSuid() != nil {
		if view.ctx.ViewSuid().INVOKER_SYMBOL() != nil {
			sqlSecurity = "INVOKER"
		}
	} else {
		return nil, errors.Errorf("view %s has no view suid field", view.name)
	}

	cols, selectExpr := "", ""
	if view.ctx.ViewTail() != nil {
		cols = view.ctx.ViewTail().ColumnInternalRefList().GetText()
		selectExpr = view.ctx.ViewTail().ViewSelect().GetText()
	} else {
		return nil, errors.Errorf("view %s has no view tail field", view.name)
	}
	// add sqlcache opts
	selectExpr = selectExpr[:6] + "SQL_NO_CACHE" + selectExpr[6:]

	newView := &viewDef{
		ctx:    view.ctx,
		name:   view.name,
		dbName: view.dbName,
	}
	newView.tempView = fmt.Sprintf("CREATE OR REPLACE DEFINER=`%s` SQL SECURITY %s VIEW `%s` %s AS %s;\n\n", definer, sqlSecurity, view.name, cols, selectExpr)
	return newView, nil
}

func (diff *diffNode) diffFunction(oldDatabase, newDatabase *databaseDef) error {
	for _, function := range newDatabase.functions {
		functionName := function.name

		oldFunction, ok := oldDatabase.functions[functionName]
		if ok {
			if !isFunctionEqual(oldFunction, function) {
				diff.dropFunctionList = append(diff.dropFunctionList, oldFunction)
			}
			delete(oldDatabase.functions, functionName)
		}
		diff.createFunctionList = append(diff.createFunctionList, function)
	}

	for _, function := range oldDatabase.functions {
		diff.dropFunctionList = append(diff.dropFunctionList, function)
	}
	return nil
}

func isFunctionEqual(old, new *functionDef) bool {
	return old.ctx.GetText() == new.ctx.GetText()
}

func (diff *diffNode) diffProcedure(oldDatabase, newDatabase *databaseDef) error {
	for _, procedure := range newDatabase.procedures {
		procedureName := procedure.name

		oldProcedure, ok := oldDatabase.procedures[procedureName]
		if ok {
			if !isProcedureEqual(oldProcedure, procedure) {
				diff.dropProcedureList = append(diff.dropProcedureList, oldProcedure)
			}
			delete(oldDatabase.functions, procedureName)
		}
		diff.createProcedureList = append(diff.createProcedureList, procedure)
	}

	for _, procedure := range oldDatabase.procedures {
		diff.dropProcedureList = append(diff.dropProcedureList, procedure)
	}
	return nil
}

func isProcedureEqual(old, new *procedureDef) bool {
	return old.ctx.GetText() == new.ctx.GetText()
}

func (diff *diffNode) diffEvent(oldSchema, newSchema *schemaDef) error {
	for _, event := range newSchema.events {
		eventName := event.name

		oldEvent, ok := oldSchema.events[eventName]
		if ok {
			if !isEventEqual(oldEvent, event) {
				diff.dropEventList = append(diff.dropEventList, oldEvent)
			}
			delete(oldSchema.events, eventName)
		}
		diff.createEventList = append(diff.createEventList, event)
	}
	for _, event := range oldSchema.events {
		diff.dropEventList = append(diff.dropEventList, event)
	}
	return nil
}

func isEventEqual(old, new *eventDef) bool {
	return old.ctx.GetText() == new.ctx.GetText()
}

func (diff *diffNode) diffTrigger(oldSchema, newSchema *schemaDef) error {
	for _, trigger := range newSchema.triggers {
		triggerName := trigger.name

		oldTrigger, ok := oldSchema.triggers[triggerName]
		if ok {
			if !isTriggerEqual(oldTrigger, trigger) {
				diff.dropTriggerList = append(diff.dropTriggerList, oldTrigger)
			}
			delete(oldSchema.events, triggerName)
		}
		diff.createTriggerList = append(diff.createTriggerList, trigger)
	}
	for _, trigger := range oldSchema.triggers {
		diff.dropTriggerList = append(diff.dropTriggerList, trigger)
	}
	return nil
}

func isTriggerEqual(old, new *triggerDef) bool {
	return old.ctx.GetText() == new.ctx.GetText()
}

func (diff *diffNode) diffColumn(oldTable, newTable *tableDef) {
	// We use a single ALTER TABLE statement to add and modify columns,
	// because we need to maintain a fixed order of these two operations.
	// This approach ensures that we can manipulate the column position as needed.
	newColumns := convertColumnMapToSortedList(newTable.columns)
	oldColumns := convertColumnMapToSortedList(oldTable.columns)
	for idx, newColumnDef := range newColumns {
		newColumnName := newColumnDef.name
		oldColumnDef, ok := oldTable.columns[newColumnName]
		if !ok {
			// create
			columnPosition := &columnPositionDef{
				tp: "FIRST",
			}
			if idx > 0 {
				columnPosition.tp = "AFTER"
				columnPosition.relativeColumn = newColumns[idx-1].name
			}
			newColumnDef.columnPosition = columnPosition
			diff.addColumnList = append(diff.addColumnList, newColumnDef)
			continue
		}
		// update
		columnPosition := &columnPositionDef{}
		if hasColumnsIntersection(oldColumns[:oldColumnDef.id], newColumns[idx+1:]) {
			if idx == 0 {
				columnPosition.tp = "FIRST"
			} else {
				columnPosition.tp = "AFTER"
				columnPosition.relativeColumn = newColumns[idx-1].name
			}
		}
		newColumnDef.columnPosition = columnPosition
		if !isColumnEqual(oldColumnDef, newColumnDef) {
			diff.modifyColumnList = append(diff.modifyColumnList, newColumnDef)
		}

		delete(oldTable.columns, newColumnName)
	}

	// drop
	for _, column := range oldTable.columns {
		diff.dropColumnList = append(diff.dropColumnList, column)
	}
}

func convertColumnMapToSortedList(columns map[string]*columnDef) (newColumns []*columnDef) {
	for _, column := range columns {
		newColumns = append(newColumns, column)
	}
	sort.Slice(newColumns, func(i, j int) bool {
		return newColumns[i].id < newColumns[j].id
	})
	return newColumns
}

// hasColumnsIntersection returns true if two column slices have column name intersection.
func hasColumnsIntersection(a, b []*columnDef) bool {
	bMap := make(map[string]bool)
	for _, col := range b {
		// MySQL column name is case insensitive.
		bMap[col.name] = true
	}
	for _, col := range a {
		// MySQL column name is case insensitive.
		if _, ok := bMap[col.name]; ok {
			return true
		}
	}
	return false
}

func (diff *diffNode) diffIndex(oldTable, newTable *tableDef) {
	// https://stackoverflow.com/questions/887590/does-dropping-a-table-in-mysql-also-drop-the-indexes
	for indexName, newIndex := range newTable.indexes {
		if oldIndex, ok := oldTable.indexes[indexName]; ok {
			if !isIndexEqual(newIndex, oldIndex) {
				diff.dropIndexList = append(diff.dropIndexList, oldIndex)
				diff.createIndexList = append(diff.createIndexList, newIndex)
			}
			delete(oldTable.indexes, indexName)
			continue
		}
		diff.createIndexList = append(diff.createIndexList, newIndex)
	}

	for _, oldIndex := range oldTable.indexes {
		diff.dropIndexList = append(diff.dropIndexList, oldIndex)
	}
}

func (diff *diffNode) buildSchemaInfo(statement string) (*databaseDef, error) {
	return diff.parseMySQLSchemaStringToSchemDef(statement)
}

// isViewEqual checks whether two views with same name are equal.
func (*diffNode) isViewEqual(old, new *viewDef) bool {
	if old.name != new.name {
		return false
	}

	if old.ctx.GetText() != new.ctx.GetText() {
		return false
	}
	return true
}

// isColumnEqual returns true if definitions of two columns with the same name are the same.
func isColumnEqual(old, new *columnDef) bool {
	if !isColumnTypesEqual(old, new) {
		return false
	}
	if !isColumnOptionsEqual(old, new) {
		return false
	}
	return true
}

func isColumnTypesEqual(old, new *columnDef) bool {
	return old.tp == new.tp
}

func isColumnOptionsEqual(old, new *columnDef) bool {
	if old.nullable != new.nullable {
		return false
	}
	if old.comment != new.comment {
		return false
	}
	if old.defaultValue != new.defaultValue {
		return false
	}
	return true
}

// isIndexEqual returns true if definitions of two indexes are the same.
func isIndexEqual(old, new *indexDef) bool {
	// CREATE [UNIQUE | FULLTEXT | SPATIAL] INDEX index_name
	// [index_type]
	// ON tbl_name (key_part,...)
	// [index_option]
	// [algorithm_option | lock_option] ...

	// MySQL index names are case insensitive.
	if !strings.EqualFold(old.name, new.name) {
		return false
	}
	if old.tp != new.tp {
		return false
	}

	if !isKeyPartEqual(old.keys, new.keys) {
		return false
	}

	if old.ctx.GetText() != new.ctx.GetText() {
		return false
	}

	return true
}

func isKeyPartEqual(old, new []string) bool {
	if len(old) != len(new) {
		return false
	}

	for idx, oldKey := range old {
		if oldKey != new[idx] {
			return false
		}
	}

	return true
}

func (diff *diffNode) deparse() (string, error) {
	var buf strings.Builder
	if err := sortAndWriteDropFunctionList(&buf, diff.dropFunctionList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropProcedureList(&buf, diff.dropProcedureList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropEventList(&buf, diff.dropEventList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropTriggerList(&buf, diff.dropTriggerList); err != nil {
		return "", err
	}

	if err := sortAndWriteDropForeignKeyList(&buf, diff.dropForeignKeyList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropCheckConstraintList(&buf, diff.dropCheckConstraintList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropPrimaryIndexList(&buf, diff.dropPrimaryKeyList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropIndexList(&buf, diff.dropIndexList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropViewList(&buf, diff.dropViewList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropTableList(&buf, diff.dropTableList); err != nil {
		return "", err
	}
	if err := sortAndWriteCreateTableList(&buf, diff.createTableList); err != nil {
		return "", err
	}
	if err := sortAndWriteAlertTableOptionList(&buf, diff.alterTableOptionList); err != nil {
		return "", err
	}

	if err := sortAndWriteAddColumnList(&buf, diff.addColumnList); err != nil {
		return "", err
	}
	if err := sortAndWriteModifyColumnList(&buf, diff.modifyColumnList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropColumnList(&buf, diff.dropColumnList); err != nil {
		return "", err
	}

	if err := sortAndWriteCreateTempViewList(&buf, diff.createTempViewList); err != nil {
		return "", err
	}

	if err := sortAndWriteCreateIndexList(&buf, diff.createIndexList); err != nil {
		return "", err
	}
	if err := sortAndWriteCreatePrimaryIndexList(&buf, diff.createPrimaryKeyList); err != nil {
		return "", err
	}
	if err := sortAndWriteDropPrimaryIndexList(&buf, diff.dropPrimaryKeyList); err != nil {
		return "", err
	}

	if err := sortAndWriteAddCheckConstraintList(&buf, diff.addCheckConstraintList); err != nil {
		return "", err
	}

	if err := sortAndWriteAddForeignKeyList(&buf, diff.addForeignKeyList); err != nil {
		return "", err
	}
	if err := sortAndWriteCreateViewList(&buf, diff.createViewList); err != nil {
		return "", err
	}

	if err := sortAndWriteCreateFunctionList(&buf, diff.createFunctionList); err != nil {
		return "", err
	}
	if err := sortAndWriteCreateProcedureList(&buf, diff.createProcedureList); err != nil {
		return "", err
	}
	if err := sortAndWriteCreateEventList(&buf, diff.createEventList); err != nil {
		return "", err
	}
	if err := sortAndWriteCreateTriggerList(&buf, diff.createTriggerList); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func sortAndWriteAlertTableOptionList(buf *strings.Builder, tableOptions []*tableOptionDef) error {
	sort.Slice(tableOptions, func(i, j int) bool {
		return tableOptions[i].tableName < tableOptions[j].tableName
	})

	for _, tableOption := range tableOptions {
		if err := writeAlertTableOptionStatement(buf, tableOption); err != nil {
			return err
		}
	}
	return nil
}

func writeAlertTableOptionStatement(buf *strings.Builder, tableOption *tableOptionDef) error {
	if _, err := buf.WriteString(tableOption.alterOption + "\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropCheckConstraintList(buf *strings.Builder, checks []*checkDef) error {
	sort.Slice(checks, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", checks[i].tableName, checks[i].name)
		c2 := fmt.Sprintf("%s.%s", checks[j].tableName, checks[j].name)
		return c1 < c2
	})

	for _, check := range checks {
		if err := writeDropCheckConstraintStatement(buf, check); err != nil {
			return err
		}
	}
	return nil
}

func writeDropCheckConstraintStatement(buf *strings.Builder, check *checkDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` DROP CHECK `%s`;\n\n", check.tableName, check.name)); err != nil {
		return err
	}
	return nil
}

func sortAndWriteAddCheckConstraintList(buf *strings.Builder, checks []*checkDef) error {
	sort.Slice(checks, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", checks[i].tableName, checks[i].name)
		c2 := fmt.Sprintf("%s.%s", checks[j].tableName, checks[j].name)
		return c1 < c2
	})

	for _, check := range checks {
		if err := writeAddCheckConstraintStatement(buf, check); err != nil {
			return err
		}
	}
	return nil
}

func writeAddCheckConstraintStatement(buf *strings.Builder, check *checkDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` ", check.tableName, check.name)); err != nil {
		return err
	}
	if _, err := buf.WriteString(check.ctx.GetText()); err != nil {
		return err
	}
	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteAddForeignKeyList(buf *strings.Builder, fks []*foreignKeyDef) error {
	sort.Slice(fks, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", fks[i].tableName, fks[i].name)
		c2 := fmt.Sprintf("%s.%s", fks[j].tableName, fks[j].name)
		return c1 < c2
	})

	for _, fk := range fks {
		if err := writeAddForeignKeyStatement(buf, fk); err != nil {
			return err
		}
	}
	return nil
}

func writeAddForeignKeyStatement(buf *strings.Builder, fk *foreignKeyDef) error {
	if _, err := buf.WriteString("ALTER TABLE " + fk.tableName + " ADD FOREIGN KEY "); err != nil {
		return err
	}
	if _, err := buf.WriteString(fk.ctx.GetText()); err != nil {
		return err
	}
	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropForeignKeyList(buf *strings.Builder, fks []*foreignKeyDef) error {
	sort.Slice(fks, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", fks[i].tableName, fks[i].name)
		c2 := fmt.Sprintf("%s.%s", fks[j].tableName, fks[j].name)
		return c1 < c2
	})

	for _, fk := range fks {
		if err := writeDropForeignKeyStatement(buf, fk); err != nil {
			return err
		}
	}
	return nil
}

func writeDropForeignKeyStatement(buf *strings.Builder, fk *foreignKeyDef) error {
	if _, err := buf.WriteString("ALTER TABLE " + fk.tableName + " DROP FOREIGN KEY "); err != nil {
		return err
	}
	if fk.ifExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(fk.name); err != nil {
		return err
	}
	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropFunctionList(buf *strings.Builder, functions []*functionDef) error {
	for _, function := range functions {
		if err := writeDropFunctionStatement(buf, function); err != nil {
			return err
		}
	}
	return nil
}

func writeDropFunctionStatement(buf *strings.Builder, function *functionDef) error {
	if _, err := buf.WriteString("DROP FUNCTION "); err != nil {
		return err
	}
	if function.ifExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(function.name + ";\n\n"); err != nil {
		return err
	}

	return nil
}

func sortAndWriteCreateFunctionList(buf *strings.Builder, functions []*functionDef) error {
	for _, function := range functions {
		if err := writeCreateFunctionStatement(buf, function); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateFunctionStatement(buf *strings.Builder, function *functionDef) error {
	var def strings.Builder
	if _, err := def.WriteString("CREATE "); err != nil {
		return err
	}
	if _, err := def.WriteString(function.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(function.ctx.GetRuleContext())); err != nil {
		return err
	}
	if _, err := def.WriteString(";;"); err != nil {
		return err
	}

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n", def.String())); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropProcedureList(buf *strings.Builder, procedures []*procedureDef) error {
	for _, procedure := range procedures {
		if err := writeDropProcedureStatement(buf, procedure); err != nil {
			return err
		}
	}
	return nil
}

func writeDropProcedureStatement(buf *strings.Builder, procedure *procedureDef) error {
	if _, err := buf.WriteString("DROP PROCEDURE "); err != nil {
		return err
	}
	if procedure.ifExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(procedure.name + ";\n\n"); err != nil {
		return err
	}

	return nil
}

func sortAndWriteCreateProcedureList(buf *strings.Builder, procedures []*procedureDef) error {
	for _, procedure := range procedures {
		if err := writeCreateProcedureStatement(buf, procedure); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateProcedureStatement(buf *strings.Builder, procedure *procedureDef) error {
	var def strings.Builder
	if _, err := def.WriteString("CREATE "); err != nil {
		return err
	}
	if _, err := def.WriteString(procedure.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(procedure.ctx.GetRuleContext())); err != nil {
		return err
	}
	if _, err := def.WriteString(";;"); err != nil {
		return err
	}

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n", def.String())); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropEventList(buf *strings.Builder, events []*eventDef) error {
	for _, event := range events {
		if err := writeDropEventStatement(buf, event); err != nil {
			return err
		}
	}
	return nil
}

func writeDropEventStatement(buf *strings.Builder, event *eventDef) error {
	if _, err := buf.WriteString("DROP EVENT "); err != nil {
		return err
	}
	if event.ifExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(event.name + ";\n\n"); err != nil {
		return err
	}

	return nil
}

func sortAndWriteCreateEventList(buf *strings.Builder, events []*eventDef) error {
	for _, event := range events {
		if err := writeCreateEventStatement(buf, event); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateEventStatement(buf *strings.Builder, event *eventDef) error {
	var def strings.Builder
	if _, err := def.WriteString("CREATE "); err != nil {
		return err
	}
	if _, err := def.WriteString(event.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(event.ctx.GetRuleContext())); err != nil {
		return err
	}
	if _, err := def.WriteString(";;"); err != nil {
		return err
	}

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n", def.String())); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropTriggerList(buf *strings.Builder, triggers []*triggerDef) error {
	for _, trigger := range triggers {
		if err := writeDropTriggerStatement(buf, trigger); err != nil {
			return err
		}
	}
	return nil
}

func writeDropTriggerStatement(buf *strings.Builder, trigger *triggerDef) error {
	if _, err := buf.WriteString("DROP TRIGGER "); err != nil {
		return err
	}
	if trigger.ifExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(trigger.name + ";\n\n"); err != nil {
		return err
	}

	return nil
}

func sortAndWriteCreateTriggerList(buf *strings.Builder, triggers []*triggerDef) error {
	for _, trigger := range triggers {
		if err := writeCreateTriggerStatement(buf, trigger); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateTriggerStatement(buf *strings.Builder, trigger *triggerDef) error {
	var def strings.Builder
	if _, err := def.WriteString("CREATE "); err != nil {
		return err
	}
	if _, err := def.WriteString(trigger.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(trigger.ctx.GetRuleContext())); err != nil {
		return err
	}
	if _, err := def.WriteString(";;"); err != nil {
		return err
	}

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n", def.String())); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropIndexList(buf *strings.Builder, indexes []*indexDef) error {
	sort.Slice(indexes, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", indexes[i].tableName, indexes[i].name)
		c2 := fmt.Sprintf("%s.%s", indexes[j].tableName, indexes[j].name)
		return c1 < c2
	})

	for _, index := range indexes {
		if err := writeDropIndexStatement(buf, index); err != nil {
			return err
		}
	}
	return nil
}

func writeDropIndexStatement(buf *strings.Builder, index *indexDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("DROP INDEX IF EXISTS `%s` ON `%s`;\n\n", index.name, index.tableName)); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropViewList(buf *strings.Builder, views []*viewDef) error {
	sort.Slice(views, func(i, j int) bool {
		return views[i].name < views[j].name
	})

	for _, view := range views {
		if err := writeDropViewStatement(buf, view); err != nil {
			return err
		}
	}
	return nil
}

func writeDropViewStatement(buf *strings.Builder, view *viewDef) error {
	if _, err := buf.WriteString("DROP VIEW "); err != nil {
		return err
	}
	if view.ifExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(view.name + ";\n\n"); err != nil {
		return err
	}

	return nil
}

func sortAndWriteDropTableList(buf *strings.Builder, ns []*tableDef) error {
	sort.Slice(ns, func(i, j int) bool {
		return ns[i].id < ns[j].id
	})

	for _, table := range ns {
		if err := writeDropTableStatement(buf, table); err != nil {
			return err
		}
	}
	return nil
}

func writeDropTableStatement(buf *strings.Builder, table *tableDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n\n", table.name)); err != nil {
		return err
	}
	return nil
}

func sortAndWriteCreateTableList(buf *strings.Builder, ns []*tableDef) error {
	sort.Slice(ns, func(i, j int) bool {
		return ns[i].id < ns[j].id
	})

	for _, table := range ns {
		if err := writeCreateTableStatement(buf, table); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateTableStatement(buf *strings.Builder, table *tableDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("CREATE " + table.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(table.ctx.GetRuleContext()))); err != nil {
		return err
	}

	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropColumnList(buf *strings.Builder, columns []*columnDef) error {
	sort.Slice(columns, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", columns[i].tableName, columns[i].name)
		c2 := fmt.Sprintf("%s.%s", columns[j].tableName, columns[j].name)
		return c1 < c2
	})

	for _, column := range columns {
		if err := writeDropColumnStatement(buf, column); err != nil {
			return err
		}
	}
	return nil
}

func writeDropColumnStatement(buf *strings.Builder, column *columnDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN IF EXISTS `%s`;\n\n", column.tableName, column.name)); err != nil {
		return err
	}
	return nil
}

func sortAndWriteAddColumnList(buf *strings.Builder, columns []*columnDef) error {
	sort.Slice(columns, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", columns[i].tableName, columns[i].name)
		c2 := fmt.Sprintf("%s.%s", columns[j].tableName, columns[j].name)
		return c1 < c2
	})

	for _, column := range columns {
		if err := writeAddColumnStatement(buf, column); err != nil {
			return err
		}
	}
	return nil
}

func writeAddColumnStatement(buf *strings.Builder, column *columnDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN IF NOT EXISTS ", column.tableName)); err != nil {
		return err
	}
	if err := column.toString(buf); err != nil {
		return err
	}
	// TODO: add column position.
	if column.columnPosition != nil {
		pos := ""
		if column.columnPosition.tp == "FIRST" {
			pos = " FIRST"
		} else if column.columnPosition.tp == "AFTER" {
			pos = fmt.Sprintf(" AFTER `%s`", column.columnPosition.relativeColumn)
		}
		if _, err := buf.WriteString(pos); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteCreateTempViewList(buf *strings.Builder, views []*viewDef) error {
	sort.Slice(views, func(i, j int) bool {
		return views[i].name < views[j].name
	})

	for _, view := range views {
		if err := writeCreateTempViewStatement(buf, view); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateTempViewStatement(buf *strings.Builder, view *viewDef) error {
	if _, err := buf.WriteString(view.tempView); err != nil {
		return err
	}

	return nil
}

func sortAndWriteCreateViewList(buf *strings.Builder, views []*viewDef) error {
	sort.Slice(views, func(i, j int) bool {
		return views[i].name < views[j].name
	})

	for _, view := range views {
		if err := writeCreateViewStatement(buf, view); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateViewStatement(buf *strings.Builder, view *viewDef) error {
	if _, err := buf.WriteString("CREATE OR REPLACE "); err != nil {
		return err
	}
	if _, err := buf.WriteString(view.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(view.ctx.GetRuleContext())); err != nil {
		return err
	}
	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}

	return nil
}

func sortAndWriteModifyColumnList(buf *strings.Builder, columns []*columnDef) error {
	sort.Slice(columns, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", columns[i].tableName, columns[i].name)
		c2 := fmt.Sprintf("%s.%s", columns[j].tableName, columns[j].name)
		return c1 < c2
	})

	for _, column := range columns {
		if err := writeModifyColumnStatement(buf, column); err != nil {
			return err
		}
	}
	return nil
}

func writeModifyColumnStatement(buf *strings.Builder, column *columnDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` ", column.tableName)); err != nil {
		return err
	}

	if _, err := buf.WriteString("MODIFY COLUMN IF EXISTS"); err != nil {
		return err
	}
	if err := column.toString(buf); err != nil {
		return err
	}
	// TODO: add column position.
	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteCreateIndexList(buf *strings.Builder, indexes []*indexDef) error {
	sort.Slice(indexes, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", indexes[i].tableName, indexes[i].name)
		c2 := fmt.Sprintf("%s.%s", indexes[j].tableName, indexes[j].name)
		return c1 < c2
	})

	for _, index := range indexes {
		if err := writeCreateIndexStatement(buf, index); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateIndexStatement(buf *strings.Builder, index *indexDef) error {
	if _, err := buf.WriteString("CREATE "); err != nil {
		return err
	}
	if _, err := buf.WriteString(index.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(index.ctx.GetRuleContext())); err != nil {
		return err
	}
	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}
	return nil
}

func sortAndWriteCreatePrimaryIndexList(buf *strings.Builder, primaryKeys []*primaryKeyDef) error {
	sort.Slice(primaryKeys, func(i, j int) bool {
		return primaryKeys[i].tableName < primaryKeys[j].tableName
	})

	for _, primaryKey := range primaryKeys {
		if err := writeCreatePrimaryIndexStatement(buf, primaryKey); err != nil {
			return err
		}
	}
	return nil
}

func writeCreatePrimaryIndexStatement(buf *strings.Builder, primary *primaryKeyDef) error {
	// todo: add more format
	cols := strings.Join(primary.columns, ",")
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD PRIMARY KEY (%s)", primary.tableName, cols)); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropPrimaryIndexList(buf *strings.Builder, primaryKeys []*primaryKeyDef) error {
	sort.Slice(primaryKeys, func(i, j int) bool {
		return primaryKeys[i].tableName < primaryKeys[j].tableName
	})

	for _, primaryKey := range primaryKeys {
		if err := writeDropPrimaryIndexStatement(buf, primaryKey); err != nil {
			return err
		}
	}
	return nil
}

func writeDropPrimaryIndexStatement(buf *strings.Builder, primary *primaryKeyDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` DROP PRIMARY KEY;\n\n", primary.tableName)); err != nil {
		return err
	}
	return nil
}

type databaseDef struct {
	name       string
	schemas    map[string]*schemaDef
	views      map[string]*viewDef
	functions  map[string]*functionDef
	procedures map[string]*procedureDef
}

func newDatabaseDef() *databaseDef {
	return &databaseDef{
		schemas:    make(map[string]*schemaDef),
		views:      make(map[string]*viewDef),
		functions:  make(map[string]*functionDef),
		procedures: make(map[string]*procedureDef),
	}
}

type viewDef struct {
	ctx      *mysql.CreateViewContext
	name     string
	dbName   string
	tempView string
	ifExists bool
}

type functionDef struct {
	ctx      *mysql.CreateFunctionContext
	name     string
	dbName   string
	ifExists bool
}

type procedureDef struct {
	ctx      *mysql.CreateProcedureContext
	name     string
	dbName   string
	ifExists bool
}

type schemaDef struct {
	name     string
	tables   map[string]*tableDef
	events   map[string]*eventDef
	triggers map[string]*triggerDef
}

func newSchemaDef() *schemaDef {
	return &schemaDef{
		tables:   make(map[string]*tableDef),
		events:   make(map[string]*eventDef),
		triggers: make(map[string]*triggerDef),
	}
}

type eventDef struct {
	ctx        *mysql.CreateEventContext
	name       string
	schemaName string
	ifExists   bool
}

type triggerDef struct {
	ctx        *mysql.CreateTriggerContext
	name       string
	schemaName string
	ifExists   bool
}

type tableDef struct {
	ctx          *mysql.CreateTableContext
	id           int
	name         string
	columns      map[string]*columnDef
	indexes      map[string]*indexDef
	foreignKeys  map[string]*foreignKeyDef
	checks       map[string]*checkDef
	tableOptions map[string]*tableOptionDef
	primaryKey   *primaryKeyDef
}

func newTableDef(id int, name string) *tableDef {
	return &tableDef{
		id:           id,
		name:         name,
		columns:      make(map[string]*columnDef),
		indexes:      make(map[string]*indexDef),
		foreignKeys:  make(map[string]*foreignKeyDef),
		checks:       make(map[string]*checkDef),
		tableOptions: make(map[string]*tableOptionDef),
	}
}

type primaryKeyDef struct {
	ctx       *mysql.TableConstraintDefContext
	columns   []string
	tableName string
}

type foreignKeyDef struct {
	ctx               *mysql.TableConstraintDefContext
	id                int
	name              string
	columns           []string
	referencedTable   string
	referencedColumns []string
	tableName         string
	ifExists          bool
}

type checkDef struct {
	ctx       *mysql.CheckConstraintContext
	id        int
	name      string
	tableName string
}

type tableOptionDef struct {
	ctx         *mysql.CreateTableOptionContext
	option      string
	alterOption string
	tableName   string
}

type indexDef struct {
	ctx       *mysql.CreateIndexContext
	id        int
	name      string
	keys      []string
	tp        string
	tableName string
}

type columnPositionDef struct {
	tp             string
	relativeColumn string
}

type columnDef struct {
	ctx            *mysql.ColumnDefinitionContext
	id             int
	name           string
	tp             string
	defaultValue   string
	comment        string
	nullable       bool
	tableName      string
	columnPosition *columnPositionDef
}

func (c *columnDef) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf("`%s` %s", c.name, c.tp)); err != nil {
		return err
	}
	if c.nullable {
		if c.defaultValue == "" {
			if _, err := buf.WriteString(" DEFAULT"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(" NULL"); err != nil {
			return err
		}
	} else {
		if _, err := buf.WriteString(" NOT NULL"); err != nil {
			return err
		}
	}
	if c.defaultValue != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" DEFAULT %s", c.defaultValue)); err != nil {
			return err
		}
	}
	if c.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", c.comment)); err != nil {
			return err
		}
	}
	return nil
}

type mysqlTransformer struct {
	*mysql.BaseMySQLParserListener

	db                  *databaseDef
	currentTable        string
	err                 error
	ignoreCaseSensitive bool
}

func (diff *diffNode) parseMySQLSchemaStringToSchemDef(schema string) (*databaseDef, error) {
	list, err := mysqlparser.ParseMySQL(schema)
	if err != nil {
		return nil, err
	}

	listener := &mysqlTransformer{
		db:                  newDatabaseDef(),
		ignoreCaseSensitive: diff.ignoreCaseSensitive,
	}
	listener.db.schemas[""] = newSchemaDef()

	for _, stmt := range list {
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	}

	return listener.db, nil
}

// EnterCreateTable is called when production createTable is entered.
func (t *mysqlTransformer) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if t.err != nil {
		return
	}

	databaseName, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if t.ignoreCaseSensitive {
		tableName = strings.ToLower(tableName)
	}
	if databaseName != "" {
		if t.db.name == "" {
			t.db.name = databaseName
		} else if t.db.name != databaseName {
			t.err = errors.New("multiple database names found: " + t.db.name + ", " + databaseName)
			return
		}
	}
	schema := t.db.schemas[""]
	if _, ok := schema.tables[tableName]; ok {
		t.err = errors.New("multiple table names found: " + tableName)
		return
	}

	schema.tables[tableName] = newTableDef(len(schema.tables), tableName)
	schema.tables[tableName].ctx = ctx
	t.currentTable = tableName
}

// ExitCreateTable is called when production createTable is exited.
func (t *mysqlTransformer) ExitCreateTable(_ *mysql.CreateTableContext) {
	t.currentTable = ""
}

// EnterCreateTableOptions is called when production createTableOptions is entered.
func (t *mysqlTransformer) EnterCreateTableOptions(ctx *mysql.CreateTableOptionsContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	for _, option := range ctx.AllCreateTableOption() {
		if tableOption, ok := option.(*mysql.CreateTableOptionContext); ok {
			newTableOption := &tableOptionDef{
				ctx:         tableOption,
				tableName:   t.currentTable,
				alterOption: fmt.Sprintf("ALTER TABLE `%s` %s", t.currentTable, tableOption.GetText()),
			}
			if tableOption.ENGINE_SYMBOL() != nil {
				newTableOption.option = "ENGINE"
			} else if tableOption.SECONDARY_ENGINE_SYMBOL() != nil {
				newTableOption.option = "SECONDARY_ENGINE"
			} else if tableOption.DefaultCharset() != nil {
				newTableOption.option = "DEFAULT CHARSET"
			} else if tableOption.DefaultCollation() != nil {
				newTableOption.option = "DEFAULT COLLATE"
			} else if tableOption.AUTO_INCREMENT_SYMBOL() != nil {
				newTableOption.option = "AUTO_INCREMENT"
			} else if tableOption.COMMENT_SYMBOL() != nil {
				newTableOption.option = "COMMENT"
			} else if tableOption.AVG_ROW_LENGTH_SYMBOL() != nil {
				newTableOption.option = "AVG_ROW_LENGTH"
			} else if tableOption.CHECKSUM_SYMBOL() != nil {
				newTableOption.option = "CHECKSUM"
			} else if tableOption.COMPRESSION_SYMBOL() != nil {
				newTableOption.option = "COMPRESSION"
			} else if tableOption.CONNECTION_SYMBOL() != nil {
				newTableOption.option = "CONNECTION"
			} else if tableOption.PASSWORD_SYMBOL() != nil {
				newTableOption.option = "PASSWORD"
			} else if tableOption.KEY_BLOCK_SIZE_SYMBOL() != nil {
				newTableOption.option = "KEY_BLOCK_SIZE"
			} else if tableOption.MAX_ROWS_SYMBOL() != nil {
				newTableOption.option = "MAX_ROWS"
			} else if tableOption.MIN_ROWS_SYMBOL() != nil {
				newTableOption.option = "MIN_ROWS"
			} else if tableOption.DELAY_KEY_WRITE_SYMBOL() != nil {
				newTableOption.option = "DELAY_KEY_WRITE"
			} else if tableOption.ROW_FORMAT_SYMBOL() != nil {
				newTableOption.option = "ROW_FORMAT"
			} else if tableOption.STATS_PERSISTENT_SYMBOL() != nil {
				newTableOption.option = "STATS_PERSISTENT"
			} else if tableOption.STATS_AUTO_RECALC_SYMBOL() != nil {
				newTableOption.option = "STATS_AUTO_RECALC"
			} else if tableOption.PACK_KEYS_SYMBOL() != nil {
				newTableOption.option = "PACK_KEYS"
			} else if tableOption.TABLESPACE_SYMBOL() != nil {
				newTableOption.option = "TABLESPACE"
			} else if tableOption.STORAGE_SYMBOL() != nil {
				newTableOption.option = "STORAGE"
			} else if tableOption.STATS_SAMPLE_PAGES_SYMBOL() != nil {
				newTableOption.option = "STATS_SAMPLE_PAGES"
			} else if tableOption.INSERT_METHOD_SYMBOL() != nil {
				newTableOption.option = "INSERT_METHOD"
			} else if tableOption.TABLE_CHECKSUM_SYMBOL() != nil {
				newTableOption.option = "TABLE_CHECKSUM"
			} else if tableOption.UNION_SYMBOL() != nil {
				newTableOption.option = "UNION"
			} else if tableOption.ENCRYPTION_SYMBOL() != nil {
				newTableOption.option = "ENCRYPTION"
			}
			table := t.db.schemas[""].tables[t.currentTable]
			table.tableOptions[newTableOption.option] = newTableOption
		}
	}
}

// EnterTableConstraintDef is called when production tableConstraintDef is entered.
func (t *mysqlTransformer) EnterTableConstraintDef(ctx *mysql.TableConstraintDefContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	if ctx.GetType_() != nil {
		switch strings.ToUpper(ctx.GetType_().GetText()) {
		case "PRIMARY":
			list := extractKeyListVariants(ctx.KeyListVariants())
			table := t.db.schemas[""].tables[t.currentTable]
			table.primaryKey = &primaryKeyDef{
				ctx:       ctx,
				columns:   list,
				tableName: t.currentTable,
			}
		case "FOREIGN":
			var name string
			if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
			} else if ctx.IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys := extractKeyList(ctx.KeyList())
			table := t.db.schemas[""].tables[t.currentTable]
			if table.foreignKeys[name] != nil {
				t.err = errors.New("multiple foreign keys found: " + name)
				return
			}
			referencedTable, referencedColumns := extractReference(ctx.References())
			fk := &foreignKeyDef{
				ctx:               ctx,
				id:                len(table.foreignKeys),
				name:              name,
				columns:           keys,
				referencedTable:   referencedTable,
				referencedColumns: referencedColumns,
				tableName:         t.currentTable,
			}
			table.foreignKeys[name] = fk
		}
	}

	if ctx.CheckConstraint() != nil {
		if constraint, ok := ctx.CheckConstraint().(*mysql.CheckConstraintContext); ok {
			var name string
			if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
			}

			table := t.db.schemas[""].tables[t.currentTable]
			ck := &checkDef{
				ctx:       constraint,
				id:        len(table.checks),
				name:      name,
				tableName: t.currentTable,
			}
			table.checks[ck.name] = ck
		}
	}
}

// extract table name and column names.
func extractReference(ctx mysql.IReferencesContext) (string, []string) {
	_, table := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if ctx.IdentifierListWithParentheses() != nil {
		columns := extractIdentifierList(ctx.IdentifierListWithParentheses().IdentifierList())
		return table, columns
	}
	return table, nil
}

// extract column names.
func extractIdentifierList(ctx mysql.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, mysqlparser.NormalizeMySQLIdentifier(identifier))
	}
	return result
}

// extract column names.
func extractKeyListVariants(ctx mysql.IKeyListVariantsContext) []string {
	if ctx.KeyList() != nil {
		return extractKeyList(ctx.KeyList())
	}
	if ctx.KeyListWithExpression() != nil {
		return extractKeyListWithExpression(ctx.KeyListWithExpression())
	}
	return nil
}

// extract column names.
func extractKeyListWithExpression(ctx mysql.IKeyListWithExpressionContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPartOrExpression() {
		if key.KeyPart() != nil {
			keyText := mysqlparser.NormalizeMySQLIdentifier(key.KeyPart().Identifier())
			result = append(result, keyText)
		} else if key.ExprWithParentheses() != nil {
			keyText := key.GetParser().GetTokenStream().GetTextFromRuleContext(key.ExprWithParentheses())
			result = append(result, keyText)
		}
	}
	return result
}

// extract column names.
func extractKeyList(ctx mysql.IKeyListContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPart() {
		keyText := mysqlparser.NormalizeMySQLIdentifier(key.Identifier())
		result = append(result, keyText)
	}
	return result
}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (t *mysqlTransformer) EnterColumnDefinition(ctx *mysql.ColumnDefinitionContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	_, _, columnName := mysqlparser.NormalizeMySQLColumnName(ctx.ColumnName())
	dataType := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.FieldDefinition().DataType())
	table := t.db.schemas[""].tables[t.currentTable]
	if _, ok := table.columns[columnName]; ok {
		t.err = errors.New("multiple column names found: " + columnName + " in table " + t.currentTable)
		return
	}
	columnState := &columnDef{
		ctx:          ctx,
		id:           len(table.columns),
		name:         columnName,
		tp:           dataType,
		defaultValue: "",
		comment:      "",
		nullable:     true,
		tableName:    t.currentTable,
	}

	for _, attribute := range ctx.FieldDefinition().AllColumnAttribute() {
		switch {
		case attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil:
			columnState.nullable = false
		case attribute.DEFAULT_SYMBOL() != nil && attribute.SERIAL_SYMBOL() == nil:
			defaultValueStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.DEFAULT_SYMBOL().GetSymbol().GetTokenIndex())
			defaultValue := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: defaultValueStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			columnState.defaultValue = defaultValue
		case attribute.COMMENT_SYMBOL() != nil:
			commentStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.COMMENT_SYMBOL().GetSymbol().GetTokenIndex())
			comment := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: commentStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			if comment != `''` && len(comment) > 2 {
				columnState.comment = comment[1 : len(comment)-1]
			}
		}
	}
	// serial meaning not null.
	if strings.ToLower(dataType) == "serial" {
		columnState.nullable = false
	}

	table.columns[columnName] = columnState
}

func nextDefaultChannelTokenIndex(tokens antlr.TokenStream, currentIndex int) int {
	for i := currentIndex + 1; i < tokens.Size(); i++ {
		if tokens.Get(i).GetChannel() == antlr.TokenDefaultChannel {
			return i
		}
	}
	return 0
}

// EnterCreateIndex is called when production createIndex is entered.
func (t *mysqlTransformer) EnterCreateIndex(ctx *mysql.CreateIndexContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	if t.ignoreCaseSensitive {
		tableName = strings.ToLower(tableName)
	}
	indexName := mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
	table, exists := t.db.schemas[""].tables[tableName]
	if !exists {
		t.err = errors.Errorf("Try to create index `%s` on table `%s`, but table not found", indexName, tableName)
		return
	}
	// Index names are always case insensitive
	if _, exists := table.indexes[strings.ToLower(indexName)]; exists {
		t.err = errors.Errorf("Try to create index `%s` on table `%s`, but index already exists", indexName, tableName)
		return
	}
	cols := extractKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	index := &indexDef{
		ctx:       ctx,
		id:        len(table.indexes),
		name:      indexName,
		keys:      cols,
		tableName: tableName,
	}

	table.indexes[strings.ToLower(indexName)] = index
	if ctx.GetType_() != nil {
		switch strings.ToUpper(ctx.GetType_().GetText()) {
		case "UNIQUE":
			index.tp = "UNIQUE"
		case "FULLTEXT":
			index.tp = "FULLTEXT"
		case "SPATIAL":
			index.tp = "SPATIAL"
		}
	}
}

// EnterCreateView is called when production createView is entered.
func (t *mysqlTransformer) EnterCreateView(ctx *mysql.CreateViewContext) {
	databaseName, viewName := mysqlparser.NormalizeMySQLViewName(ctx.ViewName())
	if t.ignoreCaseSensitive {
		viewName = strings.ToLower(viewName)
	}
	if databaseName != "" {
		if t.db.name == "" {
			t.db.name = databaseName
		} else if t.db.name != databaseName {
			t.err = errors.New("multiple database names found: " + t.db.name + ", " + databaseName)
			return
		}
	}

	t.db.views[viewName] = &viewDef{
		ctx:    ctx,
		name:   viewName,
		dbName: t.db.name,
	}
}

// EnterCreateEvent is called when production createEvent is entered.
func (t *mysqlTransformer) EnterCreateEvent(ctx *mysql.CreateEventContext) {
	schemaName, eventName := mysqlparser.NormalizeMySQLEventName(ctx.EventName())
	if t.ignoreCaseSensitive {
		eventName = strings.ToLower(eventName)
	}
	if schemaName != "" {
		if schema, ok := t.db.schemas[schemaName]; !ok {
			t.db.schemas[schemaName] = &schemaDef{
				name: schemaName,
			}
		} else if schema.name != schemaName {
			t.err = errors.New("multiple schema names found: " + schema.name + ", " + schemaName)
			return
		}
	}

	t.db.schemas[schemaName].events[eventName] = &eventDef{
		ctx:        ctx,
		name:       eventName,
		schemaName: schemaName,
	}
}

// EnterCreateTrigger is called when production createTrigger is entered.
func (t *mysqlTransformer) EnterCreateTrigger(ctx *mysql.CreateTriggerContext) {
	schemaName, triggerName := mysqlparser.NormalizeMySQLTriggerName(ctx.TriggerName())
	if t.ignoreCaseSensitive {
		triggerName = strings.ToLower(triggerName)
	}
	if schemaName != "" {
		if schema, ok := t.db.schemas[schemaName]; !ok {
			t.db.schemas[schemaName] = &schemaDef{
				name: schemaName,
			}
		} else if schema.name != schemaName {
			t.err = errors.New("multiple schema names found: " + schema.name + ", " + schemaName)
			return
		}
	}

	t.db.schemas[schemaName].triggers[triggerName] = &triggerDef{
		ctx:        ctx,
		name:       triggerName,
		schemaName: schemaName,
	}
}

// EnterCreateFunction is called when production createFunction is entered.
func (t *mysqlTransformer) EnterCreateFunction(ctx *mysql.CreateFunctionContext) {
	databaseName, functionName := mysqlparser.NormalizeMySQLFunctionName(ctx.FunctionName())
	if t.ignoreCaseSensitive {
		functionName = strings.ToLower(functionName)
	}
	if databaseName != "" {
		if t.db.name == "" {
			t.db.name = databaseName
		} else if t.db.name != databaseName {
			t.err = errors.New("multiple database names found: " + t.db.name + ", " + databaseName)
			return
		}
	}

	t.db.functions[functionName] = &functionDef{
		ctx:    ctx,
		name:   functionName,
		dbName: t.db.name,
	}
}

// EnterCreateProcedure is called when production createProcedure is entered.
func (t *mysqlTransformer) EnterCreateProcedure(ctx *mysql.CreateProcedureContext) {
	databaseName, procedureName := mysqlparser.NormalizeMySQLProcedureName(ctx.ProcedureName())
	if t.ignoreCaseSensitive {
		procedureName = strings.ToLower(procedureName)
	}
	if databaseName != "" {
		if t.db.name == "" {
			t.db.name = databaseName
		} else if t.db.name != databaseName {
			t.err = errors.New("multiple database names found: " + t.db.name + ", " + databaseName)
			return
		}
	}

	t.db.procedures[procedureName] = &procedureDef{
		ctx:    ctx,
		name:   procedureName,
		dbName: t.db.name,
	}
}
