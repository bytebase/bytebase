package tsql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	tsql "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// In fact, SQL Server is possible to create a case-sensitive database and case-insensitive database on one instance.
// https://www.webucator.com/article/how-to-check-case-sensitivity-in-sql-server/
// But by default, SQL Server is case-insensitive.

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_MSSQL, SchemaDiff)
}

type diffNode struct {
	dropConstraint []string
	dropIndex      []string
	dropColumn     []string
	dropTable      []string
	dropSchema     []string
	createSchema   []string
	createTable    []string
	addColumn      []string
	modifyColumn   []string
	addIndex       []string
	addConstraint  []string
}

func (d *diffNode) String() (string, error) {
	var buf strings.Builder
	for _, dropConstraint := range d.dropConstraint {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", dropConstraint); err != nil {
			return "", err
		}
	}
	for _, dropIndex := range d.dropIndex {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", dropIndex); err != nil {
			return "", err
		}
	}
	for _, dropColumn := range d.dropColumn {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", dropColumn); err != nil {
			return "", err
		}
	}
	for _, dropTable := range d.dropTable {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", dropTable); err != nil {
			return "", err
		}
	}
	for _, dropSchema := range d.dropSchema {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", dropSchema); err != nil {
			return "", err
		}
	}
	for _, createSchema := range d.createSchema {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", createSchema); err != nil {
			return "", err
		}
	}
	for _, createTable := range d.createTable {
		if _, err := fmt.Fprintf(&buf, "%s\n\n", createTable); err != nil {
			return "", err
		}
	}
	for _, addColumn := range d.addColumn {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", addColumn); err != nil {
			return "", err
		}
	}
	for _, modifyColumn := range d.modifyColumn {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", modifyColumn); err != nil {
			return "", err
		}
	}
	for _, addIndex := range d.addIndex {
		if _, err := fmt.Fprintf(&buf, "%s\n\n", addIndex); err != nil {
			return "", err
		}
	}
	for _, addConstraint := range d.addConstraint {
		if _, err := fmt.Fprintf(&buf, "%s;\n\n", addConstraint); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func SchemaDiff(_ base.DiffContext, oldStmt, newStmt string) (string, error) {
	oldSchemaMap, err := buildSchemaMap(oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for old statement")
	}
	newSchemaMap, err := buildSchemaMap(newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for new statement")
	}

	diff := &diffNode{}
	var newSchemas []*schemaInfo
	for _, newSchema := range newSchemaMap {
		newSchemas = append(newSchemas, newSchema)
	}
	slices.SortFunc(newSchemas, func(i, j *schemaInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, newSchema := range newSchemas {
		lowerSchema := newSchema.lowerName
		oldSchema, exists := oldSchemaMap[lowerSchema]
		if !exists {
			diff.addSchema(newSchema)
			continue
		}
		oldSchema.existsInNew = true
		if err := diff.diffSchema(oldSchema, newSchema); err != nil {
			return "", errors.Wrapf(err, "failed to diff schema %q", newSchema.name)
		}
	}
	var remainingSchemas []*schemaInfo
	for _, oldSchema := range oldSchemaMap {
		if oldSchema.existsInNew {
			continue
		}
		remainingSchemas = append(remainingSchemas, oldSchema)
	}
	slices.SortFunc(remainingSchemas, func(i, j *schemaInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, oldSchema := range remainingSchemas {
		diff.dropFullSchema(oldSchema)
		oldSchema.existsInNew = true
	}
	return diff.String()
}

func (d *diffNode) dropFullSchema(schema *schemaInfo) {
	var tables []*tableInfo
	for _, table := range schema.tableMap {
		tables = append(tables, table)
	}
	slices.SortFunc(tables, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, table := range tables {
		d.dropFullTable(schema.name, table)
	}
	d.dropSchema = append(d.dropSchema, fmt.Sprintf("DROP SCHEMA [%s]", schema.name))
}

func (d *diffNode) diffSchema(oldSchema, newSchema *schemaInfo) error {
	var newTables []*tableInfo
	for _, newTable := range newSchema.tableMap {
		newTables = append(newTables, newTable)
	}
	slices.SortFunc(newTables, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, newTable := range newTables {
		lowerTable := newTable.lowerName
		oldTable, exists := oldSchema.tableMap[lowerTable]
		if !exists {
			d.createTable = append(d.createTable, newTable.node.GetParser().GetTokenStream().GetTextFromRuleContext(newTable.node))
			continue
		}
		oldTable.existsInNew = true
		if err := d.diffTable(newSchema.name, oldTable, newTable); err != nil {
			return errors.Wrapf(err, "failed to diff table %q", newTable.name)
		}
	}
	var remainingTables []*tableInfo
	for _, oldTable := range oldSchema.tableMap {
		if oldTable.existsInNew {
			continue
		}
		remainingTables = append(remainingTables, oldTable)
	}
	slices.SortFunc(remainingTables, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, oldTable := range remainingTables {
		d.dropFullTable(oldSchema.name, oldTable)
		oldTable.existsInNew = true
	}
	return nil
}

func (d *diffNode) dropFullTable(schemaName string, table *tableInfo) {
	var indexes []*indexInfo
	for _, index := range table.indexMap {
		indexes = append(indexes, index)
	}
	slices.SortFunc(indexes, func(i, j *indexInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, index := range indexes {
		d.dropIndex = append(d.dropIndex, fmt.Sprintf("DROP INDEX [%s] ON [%s].[%s]", index.name, schemaName, table.name))
	}
	d.dropTable = append(d.dropTable, fmt.Sprintf("DROP TABLE [%s].[%s]", schemaName, table.name))
}

func (d *diffNode) diffTable(schemaName string, oldTable, newTable *tableInfo) error {
	if err := d.diffColumn(schemaName, oldTable, newTable); err != nil {
		return errors.Wrapf(err, "failed to diff column for table %q", newTable.name)
	}
	if err := d.diffConstraint(schemaName, oldTable, newTable); err != nil {
		return errors.Wrapf(err, "failed to diff constraint for table %q", newTable.name)
	}

	var newIndexes []*indexInfo
	for _, newIndex := range newTable.indexMap {
		newIndexes = append(newIndexes, newIndex)
	}
	slices.SortFunc(newIndexes, func(i, j *indexInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, newIndex := range newIndexes {
		lowerIndex := newIndex.lowerName
		oldIndex, exists := oldTable.indexMap[lowerIndex]
		if !exists {
			d.addIndex = append(d.addIndex, newIndex.node.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.node))
			continue
		}
		oldIndex.existsInNew = true
		if err := d.diffIndex(schemaName, oldTable.name, oldIndex, newIndex); err != nil {
			return errors.Wrapf(err, "failed to diff index %q", newIndex.name)
		}
	}
	var remainingIndexes []*indexInfo
	for _, oldIndex := range oldTable.indexMap {
		if oldIndex.existsInNew {
			continue
		}
		remainingIndexes = append(remainingIndexes, oldIndex)
	}
	slices.SortFunc(remainingIndexes, func(i, j *indexInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, oldIndex := range remainingIndexes {
		d.dropIndex = append(d.dropIndex, fmt.Sprintf("DROP INDEX [%s] ON [%s].[%s]", oldIndex.name, schemaName, oldTable.name))
		oldIndex.existsInNew = true
	}
	return nil
}

func (d *diffNode) diffIndex(schemaName, tableName string, oldIndex, newIndex *indexInfo) error {
	oldString := oldIndex.node.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			oldIndex.node.GetStart().GetTokenIndex(),
			oldIndex.node.Id_(0).GetStart().GetTokenIndex()-1,
		)) + oldIndex.node.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			oldIndex.node.Table_name().GetStop().GetTokenIndex()+1,
			oldIndex.node.GetStop().GetTokenIndex(),
		))
	newString := newIndex.node.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			newIndex.node.GetStart().GetTokenIndex(),
			newIndex.node.Id_(0).GetStart().GetTokenIndex()-1,
		)) + newIndex.node.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			newIndex.node.Table_name().GetStop().GetTokenIndex()+1,
			newIndex.node.GetStop().GetTokenIndex(),
		))

	if oldString != newString {
		d.dropIndex = append(d.dropIndex, fmt.Sprintf("DROP INDEX [%s] ON [%s].[%s]", oldIndex.name, schemaName, tableName))
		d.addIndex = append(d.addIndex, newIndex.node.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.node))
	}
	return nil
}

func (d *diffNode) diffConstraint(schemaName string, oldTable, newTable *tableInfo) error {
	var addConstraints []tsql.ITable_constraintContext
	var dropConstraints []string

	oldConstraintMap := buildConstraintMap(oldTable)
	for _, item := range newTable.node.Column_def_table_constraints().AllColumn_def_table_constraint() {
		constraint := item.Table_constraint()
		if constraint == nil {
			continue
		}
		if constraint.Id_(0) == nil {
			continue
		}
		_, lowerConstraint := NormalizeTSQLIdentifier(constraint.Id_(0))
		oldConstraint, ok := oldConstraintMap[lowerConstraint]
		if !ok {
			addConstraints = append(addConstraints, constraint)
			continue
		}

		if !isConstraintEqual(oldConstraint, constraint) {
			dropConstraints = append(dropConstraints, lowerConstraint)
			addConstraints = append(addConstraints, constraint)
		}
		delete(oldConstraintMap, lowerConstraint)
	}
	for _, constraint := range oldConstraintMap {
		constraintName, _ := NormalizeTSQLIdentifier(constraint.Id_(0))
		dropConstraints = append(dropConstraints, constraintName)
	}

	return d.appendConstraintDiff(schemaName, newTable.name, addConstraints, dropConstraints)
}

func (d *diffNode) appendConstraintDiff(schemaName, tableName string, addConstraints []tsql.ITable_constraintContext, dropConstraints []string) error {
	if len(addConstraints) > 0 {
		if err := d.appendAddConstraint(schemaName, tableName, addConstraints); err != nil {
			return err
		}
	}
	if len(dropConstraints) > 0 {
		if err := d.appendDropConstraint(schemaName, tableName, dropConstraints); err != nil {
			return err
		}
	}
	return nil
}

func (d *diffNode) appendDropConstraint(schemaName, tableName string, dropConstraints []string) error {
	for _, constraintName := range dropConstraints {
		var buf strings.Builder
		if _, err := fmt.Fprintf(&buf, "ALTER TABLE [%s].[%s] DROP CONSTRAINT [%s]", schemaName, tableName, constraintName); err != nil {
			return err
		}
		d.dropConstraint = append(d.dropConstraint, buf.String())
	}
	return nil
}

func (d *diffNode) appendAddConstraint(schemaName, tableName string, addConstraints []tsql.ITable_constraintContext) error {
	var buf strings.Builder

	if _, err := fmt.Fprintf(&buf, "ALTER TABLE [%s].[%s] ADD", schemaName, tableName); err != nil {
		return err
	}
	for i, addConstraint := range addConstraints {
		if i > 0 {
			if _, err := fmt.Fprintf(&buf, ","); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(&buf, "\n  %s", addConstraint.GetParser().GetTokenStream().GetTextFromRuleContext(addConstraint)); err != nil {
			return err
		}
	}
	d.addConstraint = append(d.addConstraint, buf.String())
	return nil
}

func (d *diffNode) diffColumn(schemaName string, oldTable, newTable *tableInfo) error {
	var addColumns []tsql.IColumn_definitionContext
	var modifyColumns []tsql.IColumn_definitionContext
	var dropColumns []string

	oldColumnMap := buildColumnMap(oldTable)
	for _, item := range newTable.node.Column_def_table_constraints().AllColumn_def_table_constraint() {
		column := item.Column_definition()
		if column == nil {
			continue
		}
		_, lowerColumn := NormalizeTSQLIdentifier(column.Id_())
		oldColumn, ok := oldColumnMap[lowerColumn]
		if !ok {
			addColumns = append(addColumns, column)
			continue
		}

		// Compare the column definition.
		if !isColumnEqual(oldColumn, column) {
			modifyColumns = append(modifyColumns, column)
		}
		delete(oldColumnMap, lowerColumn)
	}
	for _, column := range oldColumnMap {
		columnName, _ := NormalizeTSQLIdentifier(column.Id_())
		dropColumns = append(dropColumns, columnName)
	}

	return d.appendColumnDiff(schemaName, newTable.name, addColumns, modifyColumns, dropColumns)
}

func (d *diffNode) appendColumnDiff(schemaName, tableName string, addColumns, modifyColumns []tsql.IColumn_definitionContext, dropColumns []string) error {
	if len(addColumns) > 0 {
		if err := d.appendAddColumn(schemaName, tableName, addColumns); err != nil {
			return err
		}
	}
	if len(modifyColumns) > 0 {
		if err := d.appendModifyColumn(schemaName, tableName, modifyColumns); err != nil {
			return err
		}
	}
	if len(dropColumns) > 0 {
		if err := d.appendDropColumn(schemaName, tableName, dropColumns); err != nil {
			return err
		}
	}
	return nil
}

func (d *diffNode) appendDropColumn(schemaName, tableName string, dropColumns []string) error {
	for _, columnName := range dropColumns {
		var buf strings.Builder
		if _, err := fmt.Fprintf(&buf, "ALTER TABLE [%s].[%s] DROP COLUMN [%s]", schemaName, tableName, columnName); err != nil {
			return err
		}
		d.dropColumn = append(d.dropColumn, buf.String())
	}
	return nil
}

func (d *diffNode) appendModifyColumn(schemaName, tableName string, modifyColumns []tsql.IColumn_definitionContext) error {
	for _, modifyColumn := range modifyColumns {
		var buf strings.Builder
		if _, err := fmt.Fprintf(&buf, "ALTER TABLE [%s].[%s] ALTER COLUMN %s", schemaName, tableName, modifyColumn.GetParser().GetTokenStream().GetTextFromRuleContext(modifyColumn)); err != nil {
			return err
		}
		d.modifyColumn = append(d.modifyColumn, buf.String())
	}
	return nil
}

func (d *diffNode) appendAddColumn(schemaName, tableName string, addColumns []tsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := fmt.Fprintf(&buf, "ALTER TABLE [%s].[%s] ADD", schemaName, tableName); err != nil {
		return err
	}
	for i, addColumn := range addColumns {
		if i > 0 {
			if _, err := fmt.Fprintf(&buf, ","); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(&buf, "\n  %s", addColumn.GetParser().GetTokenStream().GetTextFromRuleContext(addColumn)); err != nil {
			return err
		}
	}
	d.addColumn = append(d.addColumn, buf.String())
	return nil
}

func isColumnEqual(oldColumn, newColumn tsql.IColumn_definitionContext) bool {
	oldString := oldColumn.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			oldColumn.Id_().GetStop().GetTokenIndex()+1,
			oldColumn.GetStop().GetTokenIndex(),
		))
	newString := newColumn.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			newColumn.Id_().GetStop().GetTokenIndex()+1,
			newColumn.GetStop().GetTokenIndex(),
		))
	return oldString == newString
}

func buildColumnMap(table *tableInfo) map[string]tsql.IColumn_definitionContext {
	m := make(map[string]tsql.IColumn_definitionContext)
	for _, item := range table.node.Column_def_table_constraints().AllColumn_def_table_constraint() {
		if column := item.Column_definition(); column != nil {
			_, lowerColumn := NormalizeTSQLIdentifier(column.Id_())
			m[lowerColumn] = column
		}
	}
	return m
}

func isConstraintEqual(oldConstraint, newConstraint tsql.ITable_constraintContext) bool {
	oldString := oldConstraint.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			oldConstraint.Id_(0).GetStop().GetTokenIndex()+1,
			oldConstraint.GetStop().GetTokenIndex(),
		))
	newString := newConstraint.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			newConstraint.Id_(0).GetStop().GetTokenIndex()+1,
			newConstraint.GetStop().GetTokenIndex(),
		))
	return oldString == newString
}

func buildConstraintMap(table *tableInfo) map[string]tsql.ITable_constraintContext {
	m := make(map[string]tsql.ITable_constraintContext)
	for _, item := range table.node.Column_def_table_constraints().AllColumn_def_table_constraint() {
		if constraint := item.Table_constraint(); constraint != nil {
			if constraint.Id_(0) == nil {
				continue
			}
			_, lowerConstraint := NormalizeTSQLIdentifier(constraint.Id_(0))
			m[lowerConstraint] = constraint
		}
	}
	return m
}

func (d *diffNode) addSchema(schema *schemaInfo) {
	d.createSchema = append(d.createSchema, schema.node.GetParser().GetTokenStream().GetTextFromRuleContext(schema.node))

	var newTables []*tableInfo
	for _, table := range schema.tableMap {
		newTables = append(newTables, table)
	}
	slices.SortFunc(newTables, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	for _, table := range newTables {
		d.createTable = append(d.createTable, table.node.GetParser().GetTokenStream().GetTextFromRuleContext(table.node))
		var newIndexes []*indexInfo
		for _, index := range table.indexMap {
			newIndexes = append(newIndexes, index)
		}
		slices.SortFunc(newIndexes, func(i, j *indexInfo) int {
			if i.id < j.id {
				return -1
			}
			if i.id > j.id {
				return 1
			}
			return 0
		})
		for _, index := range newIndexes {
			d.addIndex = append(d.addIndex, index.node.GetParser().GetTokenStream().GetTextFromRuleContext(index.node))
		}
	}
}

func buildSchemaMap(stmt string) (schemaMap, error) {
	node, err := ParseTSQL(stmt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse TSQL statement")
	}

	m := make(schemaMap)
	m[defaultSchema] = newSchemaInfo(0, defaultSchema, nil)
	listener := &buildSchemaInfoListener{
		m: m,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, node.Tree)
	return listener.m, listener.err
}

type buildSchemaInfoListener struct {
	*tsql.BaseTSqlParserListener

	m   schemaMap
	err error
}

func (l *buildSchemaInfoListener) EnterCreate_schema(ctx *tsql.Create_schemaContext) {
	if l.err != nil {
		return
	}
	if ctx.GetSchema_name() == nil {
		return
	}
	originalName, lowerName := NormalizeTSQLIdentifier(ctx.GetSchema_name())
	l.m[lowerName] = newSchemaInfo(len(l.m), originalName, ctx)
}

func (l *buildSchemaInfoListener) EnterCreate_table(ctx *tsql.Create_tableContext) {
	if l.err != nil {
		return
	}
	_, schemaName, tableName := normalizeTableNameSeparated(ctx.Table_name(), "" /* fallbackDatabase */, defaultSchema, false /* caseSensitive */)
	schemaInfo := l.m[strings.ToLower(schemaName)]
	if schemaInfo == nil {
		l.err = errors.Errorf("schema %q not found", schemaName)
		return
	}
	lowerTable := strings.ToLower(tableName)
	if _, ok := schemaInfo.tableMap[lowerTable]; ok {
		l.err = errors.Errorf("table %q already exists in schema %q", tableName, schemaName)
		return
	}
	schemaInfo.tableMap[lowerTable] = newTableInfo(len(schemaInfo.tableMap), tableName, ctx)
}

func (l *buildSchemaInfoListener) EnterCreate_index(ctx *tsql.Create_indexContext) {
	if l.err != nil {
		return
	}
	_, schemaName, tableName := normalizeTableNameSeparated(ctx.Table_name(), "" /* fallbackDatabase */, defaultSchema, false /* caseSensitive */)
	schemaInfo := l.m[strings.ToLower(schemaName)]
	if schemaInfo == nil {
		l.err = errors.Errorf("schema %q not found", schemaName)
		return
	}
	lowerTable := strings.ToLower(tableName)
	tableInfo := schemaInfo.tableMap[lowerTable]
	if tableInfo == nil {
		l.err = errors.Errorf("table %q not found in schema %q", tableName, schemaName)
		return
	}
	indexName, lowerIndex := NormalizeTSQLIdentifier(ctx.Id_(0))
	if _, ok := tableInfo.indexMap[lowerIndex]; ok {
		l.err = errors.Errorf("index %q already exists in schema %q", indexName, schemaName)
		return
	}
	tableInfo.indexMap[lowerIndex] = newIndexInfo(len(tableInfo.indexMap), indexName, ctx)
}

type schemaMap map[string]*schemaInfo
type tableMap map[string]*tableInfo
type indexMap map[string]*indexInfo

type schemaInfo struct {
	id          int
	name        string
	lowerName   string
	tableMap    tableMap
	node        *tsql.Create_schemaContext
	existsInNew bool
}

func newSchemaInfo(id int, name string, node *tsql.Create_schemaContext) *schemaInfo {
	return &schemaInfo{
		id:        id,
		name:      name,
		lowerName: strings.ToLower(name),
		tableMap:  make(tableMap),
		node:      node,
	}
}

type tableInfo struct {
	id          int
	name        string
	lowerName   string
	indexMap    indexMap
	node        *tsql.Create_tableContext
	existsInNew bool
}

func newTableInfo(id int, name string, node *tsql.Create_tableContext) *tableInfo {
	return &tableInfo{
		id:        id,
		name:      name,
		lowerName: strings.ToLower(name),
		indexMap:  make(indexMap),
		node:      node,
	}
}

type indexInfo struct {
	id          int
	name        string
	lowerName   string
	node        *tsql.Create_indexContext
	existsInNew bool
}

func newIndexInfo(id int, name string, node *tsql.Create_indexContext) *indexInfo {
	return &indexInfo{
		id:        id,
		name:      name,
		lowerName: strings.ToLower(name),
		node:      node,
	}
}
