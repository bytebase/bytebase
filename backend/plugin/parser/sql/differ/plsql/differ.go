// Package plsql provides the plsql parser plugin.
package plsql

import (
	"fmt"
	"sort"

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
	dropTable   []string
	createTable []string
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

	diff := &diffNode{}
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
		diff.diffTable(oldTable, newTable)
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

func (diff *diffNode) diffTable(oldTable, newTable *tableInfo) {
	diff.diffColumn(oldTable, newTable)
}

func (diff *diffNode) diffColumn(oldTable, newTable *tableInfo) {
	if newTable.createTable.Relational_table() == nil {
		// TODO: support object_table and xmltype_table
		return
	}

	for _, item := range newTable.createTable.Relational_table().AllRelational_property() {
		if item.Column_definition() == nil {
			continue
		}

	}
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
