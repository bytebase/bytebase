// Package plsql provides the plsql parser plugin.
package plsql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/differ"

	plsql "github.com/bytebase/plsql-parser"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

func init() {
	differ.Register(parser.Oracle, &SchemaDiffer{})
}

// SchemaDiffer is the schema differ for plsql.
type SchemaDiffer struct {
}

type diffNode struct {
	schemaName  string
	dropTable   []string
	createTable []string
	addColumn   []string
}

// SchemaDiff implements the differ.SchemaDiffer interface.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string, _ bool) (string, error) {
	oldSchemaInfo, err := buildSchemaInfo(oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for old statement")
	}
	newSchemaInfo, err := buildSchemaInfo(newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for new statement")
	}

	diff := &diffNode{
		schemaName: oldSchemaInfo.name,
	}
	var newTables []*tableInfo
	for _, table := range newSchemaInfo.tableMap {
		newTables = append(newTables, table)
	}
	sort.Slice(newTables, func(i, j int) bool {
		return newTables[i].id < newTables[j].id
	})
	for _, newTable := range newTables {
		tableName := newTable.name
		oldTable, exists := oldSchemaInfo.tableMap[tableName]
		if !exists {
			diff.createTable = append(diff.createTable, newTable.createTable.GetParser().GetTokenStream().GetTextFromRuleContext(newTable.createTable))
			continue
		}
		if err := diff.diffTable(oldTable, newTable); err != nil {
			return "", err
		}
		delete(oldSchemaInfo.tableMap, tableName)
	}

	var remainingTables []*tableInfo
	for _, table := range oldSchemaInfo.tableMap {
		remainingTables = append(remainingTables, table)
	}
	sort.Slice(remainingTables, func(i, j int) bool {
		return remainingTables[i].id < remainingTables[j].id
	})
	for _, table := range remainingTables {
		diff.dropTable = append(diff.dropTable, fmt.Sprintf(`DROP TABLE "%s"."%s";`, oldSchemaInfo.name, table.name))
	}

	return "", nil
}

func (diff *diffNode) diffTable(oldTable, newTable *tableInfo) error {
	if err := diff.diffColumn(oldTable, newTable); err != nil {
		return err
	}

	return nil
}

func buildColumnMap(table *tableInfo) map[string]plsql.IColumn_definitionContext {
	columnMap := make(map[string]plsql.IColumn_definitionContext)
	if table.createTable.Relational_table() == nil {
		return columnMap
	}
	for _, item := range table.createTable.Relational_table().AllRelational_property() {
		if item.Column_definition() == nil {
			continue
		}
		columnName := parser.PLSQLNormalizeIdentifierContext(item.Column_definition().Column_name().Identifier())
		columnMap[columnName] = item.Column_definition()
	}
	return columnMap
}

func (diff *diffNode) diffColumn(oldTable, newTable *tableInfo) error {
	if newTable.createTable.Relational_table() == nil {
		// TODO: support object_table and xmltype_table
		return nil
	}

	var addColumns []plsql.IColumn_definitionContext
	var modifyColumns []plsql.IColumn_definitionContext
	var dropColumns []string
	oldColumnMap := buildColumnMap(oldTable)
	for _, item := range newTable.createTable.Relational_table().AllRelational_property() {
		if item.Column_definition() == nil {
			continue
		}

		newColumnName := parser.PLSQLNormalizeIdentifierContext(item.Column_definition().Column_name().Identifier())
		oldColumn, ok := oldColumnMap[newColumnName]
		if !ok {
			addColumns = append(addColumns, item.Column_definition())
			continue
		}

		// Compare the column definition.
		if !isColumnEqual(oldColumn, item.Column_definition()) {
			modifyColumns = append(modifyColumns, item.Column_definition())
		}
		delete(oldColumnMap, newColumnName)
	}
	for _, column := range oldColumnMap {
		dropColumns = append(dropColumns, parser.PLSQLNormalizeIdentifierContext(column.Column_name().Identifier()))
	}

	return diff.appendColumnDiff(oldTable.name, addColumns, modifyColumns, dropColumns)
}

func (diff *diffNode) appendColumnDiff(tableName string, addColumns []plsql.IColumn_definitionContext, modifyColumns []plsql.IColumn_definitionContext, dropColumns []string) error {
	if len(addColumns) != 0 {
		if err := diff.appendAddColumn(tableName, addColumns); err != nil {
			return err
		}
	}
	if len(modifyColumns) != 0 {
		if err := diff.appendModifyColumn(tableName, modifyColumns); err != nil {
			return err
		}
	}
	if len(dropColumns) != 0 {
		if err := diff.appendDropColumn(tableName, dropColumns); err != nil {
			return err
		}
	}
	return nil
}

func (diff *diffNode) appendDropColumn(tableName string, dropColumns []string) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" DROP (`); err != nil {
		return err
	}
	for i, column := range dropColumns {
		if i != 0 {
			if _, err := buf.WriteString(`,`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n	"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	return nil
}

func (diff *diffNode) appendModifyColumn(tableName string, modifyColumns []plsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" MODIFY (`); err != nil {
		return err
	}
	for i, column := range modifyColumns {
		if i != 0 {
			if _, err := buf.WriteString(`,`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n	"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column.GetParser().GetTokenStream().GetTextFromRuleContext(column)); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	return nil
}

func (diff *diffNode) appendAddColumn(tableName string, addColumns []plsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" ADD (`); err != nil {
		return err
	}
	for i, column := range addColumns {
		if i != 0 {
			if _, err := buf.WriteString(`,`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n	"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column.GetParser().GetTokenStream().GetTextFromRuleContext(column)); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	return nil
}

func isColumnEqual(oldColumn, newColumn plsql.IColumn_definitionContext) bool {
	// TODO: compare column definition instead of text.
	oldString := oldColumn.GetParser().GetTokenStream().GetTextFromRuleContext(oldColumn)
	newString := newColumn.GetParser().GetTokenStream().GetTextFromRuleContext(newColumn)
	return oldString == newString
}

func buildSchemaInfo(statement string) (*schemaInfo, error) {
	node, _, err := parser.ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	listener := &buildSchemaInfoListener{
		schemaInfo: &schemaInfo{
			name:     "",
			tableMap: make(tableMap),
			indexMap: make(indexMap),
		},
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, node)
	if listener.err != nil {
		return nil, listener.err
	}
	return listener.schemaInfo, nil
}

type buildSchemaInfoListener struct {
	*plsql.BasePlSqlParserListener

	schemaInfo *schemaInfo
	err        error
}

// EnterCreate_table is called when production create_table is entered.
func (l *buildSchemaInfoListener) EnterCreate_table(ctx *plsql.Create_tableContext) {
	if l.err != nil {
		return
	}

	if ctx.Schema_name() != nil {
		schemaName := parser.PLSQLNormalizeIdentifierContext(ctx.Schema_name().Identifier())
		if l.schemaInfo.name == "" {
			l.schemaInfo.name = schemaName
		}
		if schemaName != l.schemaInfo.name {
			l.err = errors.Errorf("schema name mismatch: %s != %s", schemaName, l.schemaInfo.name)
			return
		}
	}
	tableName := parser.PLSQLNormalizeIdentifierContext(ctx.Table_name().Identifier())
	l.schemaInfo.tableMap[tableName] = &tableInfo{
		id:          len(l.schemaInfo.tableMap),
		name:        tableName,
		existsInNew: false,
		createTable: ctx,
	}
}

// EnterCreate_index is called when production create_index is entered.
func (l *buildSchemaInfoListener) EnterCreate_index(ctx *plsql.Create_indexContext) {
	if l.err != nil {
		return
	}

	if _, ok := ctx.GetParent().(*plsql.Unit_statementContext); !ok {
		// Skip index creation inside other statements.
		return
	}

	schema, index := parser.PLSQLNormalizeIndexName(ctx.Index_name())
	if schema != "" && l.schemaInfo.name == "" {
		l.schemaInfo.name = schema
	}
	if schema != "" && schema != l.schemaInfo.name {
		l.err = errors.Errorf("schema name mismatch: %s != %s", schema, l.schemaInfo.name)
		return
	}
	l.schemaInfo.indexMap[index] = &indexInfo{
		id:          len(l.schemaInfo.indexMap),
		name:        index,
		existsInNew: false,
		createIndex: ctx,
	}
}

type tableMap map[string]*tableInfo
type indexMap map[string]*indexInfo

type schemaInfo struct {
	id       int
	name     string
	tableMap tableMap
	indexMap indexMap
}

type tableInfo struct {
	id          int
	name        string
	existsInNew bool
	createTable plsql.ICreate_tableContext
}

type indexInfo struct {
	id          int
	name        string
	existsInNew bool
	createIndex plsql.ICreate_indexContext
}
