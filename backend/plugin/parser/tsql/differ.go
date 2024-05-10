package tsql

import (
	"fmt"
	"sort"
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
	sort.Slice(newSchemas, func(i, j int) bool {
		return newSchemas[i].id < newSchemas[j].id
	})
	for _, newSchema := range newSchemas {
		lowerSchema := newSchema.lowerName
		oldSchema, exists := oldSchemaMap[lowerSchema]
		if !exists {
			diff.addSchema(newSchema)
			continue
		}
		if err := diff.diffSchema(oldSchema, newSchema); err != nil {
			return "", errors.Wrapf(err, "failed to diff schema %q", newSchema.name)
		}
	}
}

func (d *diffNode) diffSchema(oldSchema, newSchema *schemaInfo) error {
	var newTables []*tableInfo
	for _, newTable := range newSchema.tableMap {
		newTables = append(newTables, newTable)
	}
	sort.Slice(newTables, func(i, j int) bool {
		return newTables[i].id < newTables[j].id
	})
	for _, newTable := range newTables {
		lowerTable := newTable.lowerName
		oldTable, exists := oldSchema.tableMap[lowerTable]
		if !exists {
			d.createTable = append(d.createTable, newTable.node.GetParser().GetTokenStream().GetTextFromRuleContext(newTable.node))
			continue
		}
		if err := d.diffTable(newSchema.name, oldTable, newTable); err != nil {
			return errors.Wrapf(err, "failed to diff table %q", newTable.name)
		}
	}
}

func (d *diffNode) diffTable(schemaName string, oldTable, newTable *tableInfo) error {
	if err := d.diffColumn(schemaName, oldTable, newTable); err != nil {
		return errors.Wrapf(err, "failed to diff column for table %q", newTable.name)
	}

	var newIndexes []*indexInfo
	for _, newIndex := range newTable.indexMap {
		newIndexes = append(newIndexes, newIndex)
	}
	sort.Slice(newIndexes, func(i, j int) bool {
		return newIndexes[i].id < newIndexes[j].id
	})
	for _, newIndex := range newIndexes {
		lowerIndex := newIndex.lowerName
		oldIndex, exists := oldTable.indexMap[lowerIndex]
		if !exists {
			d.addIndex = append(d.addIndex, newIndex.node.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.node))
			continue
		}
		if err := d.diffIndex(schemaName, oldIndex, newIndex); err != nil {
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
	sort.Slice(remainingIndexes, func(i, j int) bool {
		return remainingIndexes[i].id < remainingIndexes[j].id
	})
	for _, oldIndex := range remainingIndexes {
		d.dropIndex = append(d.dropIndex, fmt.Sprintf("DROP INDEX [%s] ON [%s].[%s]", oldIndex.name, schemaName, oldTable.name))
		oldIndex.existsInNew = true
	}
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

func (d *diffNode) appendAddColumn(schemaName, tableName string, addColumns []tsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := fmt.Fprintf(&buf, "ALTER TABLE [%s].[%s] ADD ", schemaName, tableName); err != nil {
		return err
	}
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

func (d *diffNode) addSchema(schema *schemaInfo) {
	d.createSchema = append(d.createSchema, schema.node.GetParser().GetTokenStream().GetTextFromRuleContext(schema.node))

	var newTables []*tableInfo
	for _, table := range schema.tableMap {
		newTables = append(newTables, table)
	}
	sort.Slice(newTables, func(i, j int) bool {
		return newTables[i].id < newTables[j].id
	})

	for _, table := range newTables {
		d.createTable = append(d.createTable, table.node.GetParser().GetTokenStream().GetTextFromRuleContext(table.node))
		var newIndexes []*indexInfo
		for _, index := range table.indexMap {
			newIndexes = append(newIndexes, index)
		}
		sort.Slice(newIndexes, func(i, j int) bool {
			return newIndexes[i].id < newIndexes[j].id
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
		m: make(schemaMap),
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
	id        int
	name      string
	lowerName string
	tableMap  tableMap
	node      *tsql.Create_schemaContext
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
