package mysql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_MYSQL, SchemaDiff)
	base.RegisterSchemaDiffFunc(storepb.Engine_OCEANBASE, SchemaDiff)
}

const (
	disableFKCheckStmt string = "SET FOREIGN_KEY_CHECKS=0;\n\n"
	enableFKCheckStmt  string = "SET FOREIGN_KEY_CHECKS=1;\n"
)

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

	dropFunctionList  []*functionDef
	dropProcedureList []*procedureDef
	dropEventList     []*eventDef
	dropTriggerList   []*triggerDef

	dropForeignKeyList      []*foreignKeyDef
	dropCheckConstraintList []*checkDef
	dropPrimaryKeyList      []*primaryKeyDef
	dropIndexList           []*indexDef
	dropIndexConstraintList []*indexConstraintDef
	dropViewList            []*viewDef
	dropTableList           []*tableDef

	createTableList      []*tableDef
	alterTableOptionList []*tableOptionDef
	addColumnList        []*columnDef
	modifyColumnList     []*columnDef
	dropColumnList       []*columnDef

	createTempViewList        []*viewDef
	createIndexList           []*indexDef
	createIndexConstraintList []*indexConstraintDef
	createPrimaryKeyList      []*primaryKeyDef
	addCheckConstraintList    []*checkDef
	addForeignKeyList         []*foreignKeyDef
	createViewList            []*viewDef

	createEventList     []*eventDef
	createTriggerList   []*triggerDef
	createFunctionList  []*functionDef
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
	diff.diffIndexConstraint(oldTable, newTable)
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
			case "ENGINE", "DEFAULT COLLATE", "DEFAULT CHARACTER SET":
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
	case "DEFAULT CHARACTER SET":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` CHARACTER SET = DEFAULT;", oldOption.tableName)
	case "DEFAULT COLLATE":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` DEFAULT COLLATE = utf8mb4_general_ci;", oldOption.tableName)
	case "AUTO_INCREMENT":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` AUTO_INCREMENT = 0;", oldOption.tableName)
	case "COMMENT":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` COMMENT = '';", oldOption.tableName)
	case "AVG_ROW_LENGTH":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` AVG_ROW_LENGTH = 0;", oldOption.tableName)
	case "CHECKSUM":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` CHECKSUM = 0;", oldOption.tableName)
	case "COMPRESSION":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` COMPRESSION = 'None';", oldOption.tableName)
	case "CONNECTION":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` CONNECTION = '';", oldOption.tableName)
	case "PASSWORD":
	case "KEY_BLOCK_SIZE":
	case "MAX_ROWS":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` MAX_ROWS = 0;", oldOption.tableName)
	case "MIN_ROWS":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` MIN_ROWS = 0;", oldOption.tableName)
	case "DELAY_KEY_WRITE":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` DELAY_KEY_WRITE = 0;", oldOption.tableName)
	case "ROW_FORMAT":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` ROW_FORMAT = DEFAULT;", oldOption.tableName)
	case "STATS_AUTO_RECALC":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` STATS_AUTO_RECALC = DEFAULT;", oldOption.tableName)
	case "PACK_KEYS":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` PACK_KEYS = DEFAULT;", oldOption.tableName)
	case "TABLESPACE":
	case "INSERT_METHOD":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` INSERT_METHOD = NO;", oldOption.tableName)
	case "TABLE_CHECKSUM":
	case "UNION":
	case "ENCRYPTION":
		oldOption.alterOption = fmt.Sprintf("ALTER TABLE `%s` ENCRYPTION = 'N';", oldOption.tableName)
	}
	return oldOption
}

// isTableOptionValEqual compare the two table options value, if they are equal, returns true.
// Caller need to ensure the two table options are not nil and the type is the same.
func isTableOptionEqual(oldOption, newOption *tableOptionDef) bool {
	if oldOption.option == "DEFAULT CHARACTER SET" {
		return oldOption.ctx.DefaultCharset().CharsetName().GetText() == newOption.ctx.DefaultCharset().CharsetName().GetText()
	}
	return oldOption.ctx.GetText() == newOption.ctx.GetText()
}

func (diff *diffNode) diffPrimaryKey(oldTable, newTable *tableDef) {
	oldPrimaryKey := oldTable.primaryKey
	newPrimaryKey := newTable.primaryKey
	if oldPrimaryKey == nil && newPrimaryKey == nil {
		return
	} else if oldPrimaryKey != nil && newPrimaryKey == nil {
		diff.dropPrimaryKeyList = append(diff.dropPrimaryKeyList, oldPrimaryKey)
		return
	} else if oldPrimaryKey == nil && newPrimaryKey != nil {
		diff.createPrimaryKeyList = append(diff.createPrimaryKeyList, newPrimaryKey)
		return
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
	for _, foreignKey := range oldTable.foreignKeys {
		diff.dropForeignKeyList = append(diff.dropForeignKeyList, foreignKey)
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
	for _, view := range newDatabase.schemas[""].views {
		viewName := view.name
		if diff.ignoreCaseSensitive {
			viewName = strings.ToLower(viewName)
		}

		oldView, ok := oldDatabase.schemas[""].views[viewName]
		if ok {
			if !diff.isViewEqual(view, oldView) {
				diff.createViewList = append(diff.createViewList, view)
			}
			// We should delete the view in the oldViewMap, because we will drop the all views in the oldViewMap explicitly at last.
			delete(oldDatabase.schemas[""].views, viewName)
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
	for _, view := range oldDatabase.schemas[""].views {
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
	algorithm := "UNDEFINED"
	if view.ctx.ViewReplaceOrAlgorithm() != nil {
		if algo := view.ctx.ViewReplaceOrAlgorithm().ViewAlgorithm(); algo != nil {
			algorithm = algo.GetAlgorithm().GetText()
		}
	}
	definer := "CURRENT_USER"
	if view.ctx.DefinerClause() != nil {
		definer = view.ctx.DefinerClause().GetText()
	}
	sqlSecurity := "DEFINER"
	if view.ctx.ViewSuid() != nil {
		if view.ctx.ViewSuid().INVOKER_SYMBOL() != nil {
			sqlSecurity = "INVOKER"
		}
	}
	var selectClause strings.Builder
	if _, err := selectClause.WriteString("SELECT SQL_NO_CACHE"); err != nil {
		return nil, err
	}
	for idx, column := range view.columns {
		if idx > 0 {
			if _, err := selectClause.WriteString(","); err != nil {
				return nil, err
			}
		}
		if _, err := selectClause.WriteString(fmt.Sprintf(" 1 AS `%s`", column)); err != nil {
			return nil, err
		}
	}

	cols := ""
	if view.ctx.ViewTail() != nil {
		if list := view.ctx.ViewTail().ColumnInternalRefList(); list != nil {
			cols = list.GetParser().GetTokenStream().GetTextFromRuleContext(list.GetRuleContext())
		}
	}
	var viewSQL string
	if cols == "" {
		viewSQL = fmt.Sprintf("CREATE OR REPLACE ALGORITHM=%s DEFINER=%s SQL SECURITY %s VIEW `%s` AS %s;\n\n", algorithm, definer, sqlSecurity, view.name, selectClause.String())
	} else {
		viewSQL = fmt.Sprintf("CREATE OR REPLACE ALGORITHM=%s DEFINER=%s SQL SECURITY %s VIEW `%s` %s AS %s;\n\n", algorithm, definer, sqlSecurity, view.name, cols, selectClause.String())
	}

	newView := &viewDef{
		ctx:    view.ctx,
		name:   view.name,
		dbName: view.dbName,
	}
	newView.tempView = viewSQL
	return newView, nil
}

func (diff *diffNode) diffFunction(oldDatabase, newDatabase *databaseDef) error {
	for _, function := range newDatabase.schemas[""].functions {
		functionName := function.name

		oldFunction, ok := oldDatabase.schemas[""].functions[functionName]
		if ok {
			if !isFunctionEqual(oldFunction, function) {
				diff.dropFunctionList = append(diff.dropFunctionList, oldFunction)
			}
			delete(oldDatabase.schemas[""].functions, functionName)
		}
		diff.createFunctionList = append(diff.createFunctionList, function)
	}

	for _, function := range oldDatabase.schemas[""].functions {
		diff.dropFunctionList = append(diff.dropFunctionList, function)
	}
	return nil
}

func isFunctionEqual(old, new *functionDef) bool {
	return old.ctx.GetText() == new.ctx.GetText()
}

func (diff *diffNode) diffProcedure(oldDatabase, newDatabase *databaseDef) error {
	for _, procedure := range newDatabase.schemas[""].procedures {
		procedureName := procedure.name

		oldProcedure, ok := oldDatabase.schemas[""].procedures[procedureName]
		if ok {
			if !isProcedureEqual(oldProcedure, procedure) {
				diff.dropProcedureList = append(diff.dropProcedureList, oldProcedure)
			}
			delete(oldDatabase.schemas[""].functions, procedureName)
		}
		diff.createProcedureList = append(diff.createProcedureList, procedure)
	}

	for _, procedure := range oldDatabase.schemas[""].procedures {
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
			delete(oldSchema.triggers, triggerName)
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

func (diff *diffNode) diffIndexConstraint(oldTable, newTable *tableDef) {
	// https://stackoverflow.com/questions/887590/does-dropping-a-table-in-mysql-also-drop-the-indexes
	for indexName, newIndex := range newTable.indexConstraints {
		if oldIndex, ok := oldTable.indexConstraints[indexName]; ok {
			if !isIndexConstraintEqual(newIndex, oldIndex) {
				diff.dropIndexConstraintList = append(diff.dropIndexConstraintList, oldIndex)
				diff.createIndexConstraintList = append(diff.createIndexConstraintList, newIndex)
			}
			delete(oldTable.indexConstraints, indexName)
			continue
		}
		diff.createIndexConstraintList = append(diff.createIndexConstraintList, newIndex)
	}

	for _, oldIndex := range oldTable.indexConstraints {
		diff.dropIndexConstraintList = append(diff.dropIndexConstraintList, oldIndex)
	}
}

func isIndexConstraintEqual(new, old *indexConstraintDef) bool {
	if old.name != new.name {
		return false
	}

	if old.ctx.GetText() != new.ctx.GetText() {
		return false
	}
	return true
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
	// column name
	return old.ctx.GetText() == new.ctx.GetText()
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
	if err := sortAndWriteDropIndexConstraintList(&buf, diff.dropIndexConstraintList); err != nil {
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

	if err := sortAndWriteCreateIndexConstraintList(&buf, diff.createIndexConstraintList); err != nil {
		return "", err
	}

	if err := sortAndWriteCreatePrimaryIndexList(&buf, diff.createPrimaryKeyList); err != nil {
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

	text := buf.String()
	if len(text) > 0 {
		return fmt.Sprintf("%s%s%s", disableFKCheckStmt, buf.String(), enableFKCheckStmt), nil
	}
	return "", nil
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
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD ", check.tableName)); err != nil {
		return err
	}
	if _, err := buf.WriteString(check.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(check.ctx.GetRuleContext())); err != nil {
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
	if _, err := buf.WriteString("ALTER TABLE " + fk.tableName + " ADD "); err != nil {
		return err
	}
	if _, err := buf.WriteString(fk.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(fk.ctx.GetRuleContext())); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`;\n\n", fk.tableName, fk.name)); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("DROP FUNCTION IF EXISTS `%s`;\n\n", function.name)); err != nil {
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

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n\n", def.String())); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("DROP PROCEDURE IF EXISTS `%s`;\n\n", procedure.name)); err != nil {
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

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n\n", def.String())); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("DROP EVENT IF EXISTS `%s`;\n\n", event.name)); err != nil {
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

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n\n", def.String())); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("DROP TRIGGER IF EXISTS `%s`;\n\n", trigger.name)); err != nil {
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

	if _, err := buf.WriteString(fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n\n", def.String())); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("DROP INDEX `%s` ON `%s`;\n\n", index.name, index.tableName)); err != nil {
		return err
	}
	return nil
}

func sortAndWriteDropIndexConstraintList(buf *strings.Builder, indexes []*indexConstraintDef) error {
	sort.Slice(indexes, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", indexes[i].tableName, indexes[i].name)
		c2 := fmt.Sprintf("%s.%s", indexes[j].tableName, indexes[j].name)
		return c1 < c2
	})

	for _, index := range indexes {
		if err := writeDropIndexConstraintStatement(buf, index); err != nil {
			return err
		}
	}
	return nil
}

func writeDropIndexConstraintStatement(buf *strings.Builder, index *indexConstraintDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("DROP INDEX `%s` ON `%s`;\n\n", index.name, index.tableName)); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("DROP VIEW IF EXISTS `%s`;\n\n", view.name)); err != nil {
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
		return ns[i].name < ns[j].name
	})

	for _, table := range ns {
		if err := writeCreateTableStatement(buf, table); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateTableStatement(buf *strings.Builder, table *tableDef) error {
	stmt := fmt.Sprintf("CREATE %s;\n\n", table.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(table.ctx.GetRuleContext()))
	if stmt[0:12] == "CREATE TABLE" {
		stmt = stmt[0:12] + " IF NOT EXISTS" + stmt[12:]
	}

	if _, err := buf.WriteString(stmt); err != nil {
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
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`;\n\n", column.tableName, column.name)); err != nil {
		return err
	}
	return nil
}

func sortAndWriteAddColumnList(buf *strings.Builder, columns []*columnDef) error {
	// we do not sort the columns here for maintaining the relative position between columns;
	for _, column := range columns {
		if err := writeAddColumnStatement(buf, column); err != nil {
			return err
		}
	}
	return nil
}

func writeAddColumnStatement(buf *strings.Builder, column *columnDef) error {
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN ", column.tableName)); err != nil {
		return err
	}
	if _, err := buf.WriteString(column.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(column.ctx.GetRuleContext())); err != nil {
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
	algorithm := "UNDEFINED"
	if view.ctx.ViewReplaceOrAlgorithm() != nil {
		if algo := view.ctx.ViewReplaceOrAlgorithm().ViewAlgorithm(); algo != nil {
			algorithm = algo.GetAlgorithm().GetText()
		}
	}

	definer := "CURRENT_USER"
	if view.ctx.DefinerClause() != nil {
		definer = view.ctx.DefinerClause().GetText()
	}
	sqlSecurity := "DEFINER"
	if view.ctx.ViewSuid() != nil {
		if view.ctx.ViewSuid().INVOKER_SYMBOL() != nil {
			sqlSecurity = "INVOKER"
		}
	}

	cols, selectExpr := "", ""
	if view.ctx.ViewTail() != nil {
		if list := view.ctx.ViewTail().ColumnInternalRefList(); list != nil {
			cols = list.GetParser().GetTokenStream().GetTextFromRuleContext(list.GetRuleContext())
		}
		if viewSelect := view.ctx.ViewTail().ViewSelect(); viewSelect != nil {
			selectExpr = viewSelect.GetParser().GetTokenStream().GetTextFromRuleContext(viewSelect.GetRuleContext())
		}
	} else {
		return errors.Errorf("view %s has no select expr", view.name)
	}
	var viewSQL string
	if cols == "" {
		viewSQL = fmt.Sprintf("CREATE OR REPLACE ALGORITHM=%s DEFINER=%s SQL SECURITY %s VIEW `%s` AS %s;\n\n", algorithm, definer, sqlSecurity, view.name, selectExpr)
	} else {
		viewSQL = fmt.Sprintf("CREATE OR REPLACE ALGORITHM=%s DEFINER=%s SQL SECURITY %s VIEW `%s` %s AS %s;\n\n", algorithm, definer, sqlSecurity, view.name, cols, selectExpr)
	}

	if _, err := buf.WriteString(viewSQL); err != nil {
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

	if _, err := buf.WriteString("MODIFY COLUMN "); err != nil {
		return err
	}

	if _, err := buf.WriteString(column.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(column.ctx.GetRuleContext())); err != nil {
		return err
	}

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

func sortAndWriteCreateIndexConstraintList(buf *strings.Builder, indexes []*indexConstraintDef) error {
	sort.Slice(indexes, func(i, j int) bool {
		c1 := fmt.Sprintf("%s.%s", indexes[i].tableName, indexes[i].name)
		c2 := fmt.Sprintf("%s.%s", indexes[j].tableName, indexes[j].name)
		return c1 < c2
	})

	for _, index := range indexes {
		if err := writeCreateIndexConstraintStatement(buf, index); err != nil {
			return err
		}
	}
	return nil
}

func writeCreateIndexConstraintStatement(buf *strings.Builder, index *indexConstraintDef) error {
	indexType := ""
	if nameAndType := index.ctx.IndexNameAndType(); nameAndType != nil {
		indexType = nameAndType.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: nameAndType.IndexName().GetStop().GetTokenIndex() + 1,
			Stop:  nameAndType.GetStop().GetTokenIndex(),
		})
	}
	keyList := index.ctx.KeyListVariants().GetParser().GetTokenStream().GetTextFromRuleContext(index.ctx.KeyListVariants().GetRuleContext())
	indexOption := index.ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: index.ctx.KeyListVariants().GetStop().GetTokenIndex() + 1,
		Stop:  index.ctx.GetStop().GetTokenIndex(),
	})
	if _, err := buf.WriteString(fmt.Sprintf("CREATE INDEX `%s` ON `%s`%s%s%s;\n\n", index.name, index.tableName, keyList, indexType, indexOption)); err != nil {
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
	keyList := primary.ctx.KeyListVariants().GetParser().GetTokenStream().GetTextFromRuleContext(primary.ctx.KeyListVariants().GetRuleContext())
	indexOption := primary.ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: primary.ctx.KeyListVariants().GetStop().GetTokenIndex() + 1,
		Stop:  primary.ctx.GetStop().GetTokenIndex(),
	})
	if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD PRIMARY KEY %s%s;\n\n", primary.tableName, keyList, indexOption)); err != nil {
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
	name    string
	schemas map[string]*schemaDef
}

func newDatabaseDef() *databaseDef {
	return &databaseDef{
		schemas: make(map[string]*schemaDef),
	}
}

type viewDef struct {
	ctx      *mysql.CreateViewContext
	name     string
	dbName   string
	tempView string
	columns  []string
}

type functionDef struct {
	ctx  *mysql.CreateFunctionContext
	name string
}

type procedureDef struct {
	ctx  *mysql.CreateProcedureContext
	name string
}

type schemaDef struct {
	name string
	// todo: check for duplicate names.
	tables     map[string]*tableDef
	views      map[string]*viewDef
	events     map[string]*eventDef
	triggers   map[string]*triggerDef
	functions  map[string]*functionDef
	procedures map[string]*procedureDef
}

func newSchemaDef() *schemaDef {
	return &schemaDef{
		tables:     make(map[string]*tableDef),
		views:      make(map[string]*viewDef),
		events:     make(map[string]*eventDef),
		triggers:   make(map[string]*triggerDef),
		functions:  make(map[string]*functionDef),
		procedures: make(map[string]*procedureDef),
	}
}

type eventDef struct {
	ctx  *mysql.CreateEventContext
	name string
}

type triggerDef struct {
	ctx  *mysql.CreateTriggerContext
	name string
}

type tableDef struct {
	ctx              *mysql.CreateTableContext
	id               int
	name             string
	columns          map[string]*columnDef
	indexes          map[string]*indexDef
	indexConstraints map[string]*indexConstraintDef
	foreignKeys      map[string]*foreignKeyDef
	checks           map[string]*checkDef
	tableOptions     map[string]*tableOptionDef
	primaryKey       *primaryKeyDef
}

func newTableDef(id int, name string) *tableDef {
	return &tableDef{
		id:               id,
		name:             name,
		columns:          make(map[string]*columnDef),
		indexes:          make(map[string]*indexDef),
		indexConstraints: make(map[string]*indexConstraintDef),
		foreignKeys:      make(map[string]*foreignKeyDef),
		checks:           make(map[string]*checkDef),
		tableOptions:     make(map[string]*tableOptionDef),
	}
}

type primaryKeyDef struct {
	ctx       *mysql.TableConstraintDefContext
	columns   []string
	tableName string
}

type indexConstraintDef struct {
	ctx       *mysql.TableConstraintDefContext
	name      string
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
}

type checkDef struct {
	ctx       *mysql.TableConstraintDefContext
	id        int
	name      string
	enforced  bool
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
	visible        bool
	tableName      string
	columnPosition *columnPositionDef
}

type mysqlTransformer struct {
	*mysql.BaseMySQLParserListener

	db                  *databaseDef
	currentTable        string
	currView            string
	err                 error
	ignoreCaseSensitive bool
}

func (diff *diffNode) parseMySQLSchemaStringToSchemDef(schema string) (*databaseDef, error) {
	list, err := ParseMySQL(schema)
	if err != nil {
		return nil, err
	}

	listener := &mysqlTransformer{
		db:                  newDatabaseDef(),
		ignoreCaseSensitive: diff.ignoreCaseSensitive,
	}
	listener.db.schemas[""] = newSchemaDef()
	listener.db.schemas[""].name = ""

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

	databaseName, tableName := NormalizeMySQLTableName(ctx.TableName())
	if t.ignoreCaseSensitive {
		databaseName = strings.ToLower(databaseName)
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
				alterOption: fmt.Sprintf("ALTER TABLE `%s` %s;", t.currentTable, tableOption.GetParser().GetTokenStream().GetTextFromRuleContext(tableOption.GetRuleContext())),
			}

			switch {
			case tableOption.ENGINE_SYMBOL() != nil:
				newTableOption.option = "ENGINE"
			case tableOption.SECONDARY_ENGINE_SYMBOL() != nil:
				newTableOption.option = "SECONDARY_ENGINE"
			case tableOption.DefaultCharset() != nil:
				newTableOption.option = "DEFAULT CHARACTER SET"
			case tableOption.DefaultCollation() != nil:
				newTableOption.option = "DEFAULT COLLATE"
			case tableOption.AUTO_INCREMENT_SYMBOL() != nil:
				newTableOption.option = "AUTO_INCREMENT"
			case tableOption.COMMENT_SYMBOL() != nil:
				newTableOption.option = "COMMENT"
			case tableOption.AVG_ROW_LENGTH_SYMBOL() != nil:
				newTableOption.option = "AVG_ROW_LENGTH"
			case tableOption.CHECKSUM_SYMBOL() != nil:
				newTableOption.option = "CHECKSUM"
			case tableOption.COMPRESSION_SYMBOL() != nil:
				newTableOption.option = "COMPRESSION"
			case tableOption.CONNECTION_SYMBOL() != nil:
				newTableOption.option = "CONNECTION"
			case tableOption.PASSWORD_SYMBOL() != nil:
				newTableOption.option = "PASSWORD"
			case tableOption.KEY_BLOCK_SIZE_SYMBOL() != nil:
				newTableOption.option = "KEY_BLOCK_SIZE"
			case tableOption.MAX_ROWS_SYMBOL() != nil:
				newTableOption.option = "MAX_ROWS"
			case tableOption.MIN_ROWS_SYMBOL() != nil:
				newTableOption.option = "MIN_ROWS"
			case tableOption.DELAY_KEY_WRITE_SYMBOL() != nil:
				newTableOption.option = "DELAY_KEY_WRITE"
			case tableOption.ROW_FORMAT_SYMBOL() != nil:
				newTableOption.option = "ROW_FORMAT"
			case tableOption.STATS_PERSISTENT_SYMBOL() != nil:
				newTableOption.option = "STATS_PERSISTENT"
			case tableOption.STATS_AUTO_RECALC_SYMBOL() != nil:
				newTableOption.option = "STATS_AUTO_RECALC"
			case tableOption.PACK_KEYS_SYMBOL() != nil:
				newTableOption.option = "PACK_KEYS"
			case tableOption.TABLESPACE_SYMBOL() != nil:
				newTableOption.option = "TABLESPACE"
			case tableOption.STORAGE_SYMBOL() != nil:
				newTableOption.option = "STORAGE"
			case tableOption.STATS_SAMPLE_PAGES_SYMBOL() != nil:
				newTableOption.option = "STATS_SAMPLE_PAGES"
			case tableOption.INSERT_METHOD_SYMBOL() != nil:
				newTableOption.option = "INSERT_METHOD"
			case tableOption.TABLE_CHECKSUM_SYMBOL() != nil:
				newTableOption.option = "TABLE_CHECKSUM"
			case tableOption.UNION_SYMBOL() != nil:
				newTableOption.option = "UNION"
			case tableOption.ENCRYPTION_SYMBOL() != nil:
				newTableOption.option = "ENCRYPTION"
			case tableOption.DATA_SYMBOL() != nil && tableOption.DIRECTORY_SYMBOL() != nil:
				newTableOption.option = "DATA DIRECTORY"
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

	table := t.db.schemas[""].tables[t.currentTable]
	if ctx.GetType_() != nil {
		switch strings.ToUpper(ctx.GetType_().GetText()) {
		case "PRIMARY":
			list := extractKeyListVariants(ctx.KeyListVariants())
			table.primaryKey = &primaryKeyDef{
				ctx:       ctx,
				columns:   list,
				tableName: t.currentTable,
			}
		case "FOREIGN":
			var name string
			if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
				name = NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
			} else if ctx.IndexName() != nil {
				name = NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys := extractKeyList(ctx.KeyList())
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
		case "KEY", "INDEX", "UNIQUE":
			indexName := strings.ToLower(ctx.IndexNameAndType().IndexName().GetText())
			if indexName != `''` && len(indexName) > 2 {
				indexName = indexName[1 : len(indexName)-1]
			}
			index := &indexConstraintDef{
				ctx:       ctx,
				name:      indexName,
				tableName: t.currentTable,
			}
			table.indexConstraints[index.name] = index
		case "FULLTEXT", "SPATIAL":
			indexName := strings.ToLower(ctx.IndexName().GetText())
			if indexName != `''` && len(indexName) > 2 {
				indexName = indexName[1 : len(indexName)-1]
			}
			index := &indexConstraintDef{
				ctx:       ctx,
				name:      indexName,
				tableName: t.currentTable,
			}
			table.indexConstraints[index.name] = index
		}
	}

	if ctx.CheckConstraint() != nil {
		if _, ok := ctx.CheckConstraint().(*mysql.CheckConstraintContext); ok {
			var name string
			if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
				name = NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
			}

			enforced := true
			if ctx.ConstraintEnforcement() != nil {
				if ctx.ConstraintEnforcement().NOT_SYMBOL() != nil {
					enforced = false
				}
			}

			table := t.db.schemas[""].tables[t.currentTable]
			ck := &checkDef{
				ctx:       ctx,
				id:        len(table.checks),
				name:      name,
				enforced:  enforced,
				tableName: t.currentTable,
			}
			table.checks[ck.name] = ck
		}
	}
}

// extract table name and column names.
func extractReference(ctx mysql.IReferencesContext) (string, []string) {
	_, table := NormalizeMySQLTableRef(ctx.TableRef())
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
		result = append(result, NormalizeMySQLIdentifier(identifier))
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
			keyText := NormalizeMySQLIdentifier(key.KeyPart().Identifier())
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
		keyText := NormalizeMySQLIdentifier(key.Identifier())
		result = append(result, keyText)
	}
	return result
}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (t *mysqlTransformer) EnterColumnDefinition(ctx *mysql.ColumnDefinitionContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	_, _, columnName := NormalizeMySQLColumnName(ctx.ColumnName())
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
		visible:      true,
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
		case attribute.INVISIBLE_SYMBOL() != nil:
			columnState.visible = false
		}
	}
	// todo: need handle more types.
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
	_, tableName := NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	if t.ignoreCaseSensitive {
		tableName = strings.ToLower(tableName)
	}
	indexName := NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
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
	if _, exists := table.indexConstraints[strings.ToLower(indexName)]; exists {
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
	_, viewName := NormalizeMySQLViewName(ctx.ViewName())
	t.currView = viewName
	if t.ignoreCaseSensitive {
		viewName = strings.ToLower(viewName)
	}

	t.db.schemas[""].views[viewName] = &viewDef{
		ctx:  ctx,
		name: viewName,
	}
}

func (t *mysqlTransformer) ExitCreateView(ctx *mysql.CreateViewContext) {
	if t.err != nil {
		return
	}

	view := t.db.schemas[""].views[t.currView]

	var tableSchemaList []base.TableSchema
	for _, table := range t.db.schemas[""].tables {
		var columnList []base.ColumnInfo
		for _, col := range table.columns {
			columnInfo := base.ColumnInfo{
				Name: col.name,
			}
			columnList = append(columnList, columnInfo)
		}
		tableSchema := base.TableSchema{
			Name:       table.name,
			ColumnList: columnList,
		}
		tableSchemaList = append(tableSchemaList, tableSchema)
	}
	var viewSchemaList []base.ViewSchema
	for _, view := range t.db.schemas[""].views {
		viewSchema := base.ViewSchema{
			Name:       view.name,
			Definition: fmt.Sprintf("CREATE %s;", view.ctx.GetParser().GetTokenStream().GetTextFromRuleContext(view.ctx.GetRuleContext())),
		}
		viewSchemaList = append(viewSchemaList, viewSchema)
	}
	schemaList := []base.SchemaSchema{
		{
			Name:      "",
			TableList: tableSchemaList,
			ViewList:  viewSchemaList,
		},
	}

	schemaInfo := &base.SensitiveSchemaInfo{
		DatabaseList: []base.DatabaseSchema{
			{
				Name:       t.db.name,
				SchemaList: schemaList,
			},
		},
	}

	extractor := &fieldExtractor{
		currentDatabase: view.dbName,
		schemaInfo:      schemaInfo,
	}

	fields, err := extractor.mysqlExtractCreateView(ctx)
	if err != nil {
		t.err = err
		return
	}

	var result []string
	for _, field := range fields {
		result = append(result, field.Name)
	}
	// the column order of create view is decided by create view statement.
	// we get columns here only for create temp view.
	// only for test-case.
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	view.columns = result
}

// EnterCreateEvent is called when production createEvent is entered.
func (t *mysqlTransformer) EnterCreateEvent(ctx *mysql.CreateEventContext) {
	_, eventName := NormalizeMySQLEventName(ctx.EventName())
	if t.ignoreCaseSensitive {
		eventName = strings.ToLower(eventName)
	}

	t.db.schemas[""].events[eventName] = &eventDef{
		ctx:  ctx,
		name: eventName,
	}
}

// EnterCreateTrigger is called when production createTrigger is entered.
func (t *mysqlTransformer) EnterCreateTrigger(ctx *mysql.CreateTriggerContext) {
	_, triggerName := NormalizeMySQLTriggerName(ctx.TriggerName())
	if t.ignoreCaseSensitive {
		triggerName = strings.ToLower(triggerName)
	}

	t.db.schemas[""].triggers[triggerName] = &triggerDef{
		ctx:  ctx,
		name: triggerName,
	}
}

// EnterCreateFunction is called when production createFunction is entered.
func (t *mysqlTransformer) EnterCreateFunction(ctx *mysql.CreateFunctionContext) {
	_, functionName := NormalizeMySQLFunctionName(ctx.FunctionName())
	if t.ignoreCaseSensitive {
		functionName = strings.ToLower(functionName)
	}

	t.db.schemas[""].functions[functionName] = &functionDef{
		ctx:  ctx,
		name: functionName,
	}
}

// EnterCreateProcedure is called when production createProcedure is entered.
func (t *mysqlTransformer) EnterCreateProcedure(ctx *mysql.CreateProcedureContext) {
	_, procedureName := NormalizeMySQLProcedureName(ctx.ProcedureName())
	if t.ignoreCaseSensitive {
		procedureName = strings.ToLower(procedureName)
	}

	t.db.schemas[""].procedures[procedureName] = &procedureDef{
		ctx:  ctx,
		name: procedureName,
	}
}
