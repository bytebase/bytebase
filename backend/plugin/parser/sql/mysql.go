// Package parser is the parser for SQL statement.
package parser

import (
	"errors"
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// MySQLValidateForEditor validates the given SQL statement for editor.
func MySQLValidateForEditor(tree antlr.Tree) error {
	l := &mysqlValidateForEditorListener{
		validate: true,
	}

	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return errors.New("only support SELECT sql statement")
	}
	return nil
}

type mysqlValidateForEditorListener struct {
	*parser.BaseMySQLParserListener

	validate bool
}

// EnterQuery is called when production query is entered.
func (l *mysqlValidateForEditorListener) EnterQuery(ctx *parser.QueryContext) {
	if ctx.BeginWork() != nil {
		l.validate = false
	}
}

// EnterSimpleStatement is called when production simpleStatement is entered.
func (l *mysqlValidateForEditorListener) EnterSimpleStatement(ctx *parser.SimpleStatementContext) {
	if ctx.SelectStatement() == nil && ctx.UtilityStatement() == nil {
		l.validate = false
	}
}

// EnterUtilityStatement is called when production utilityStatement is entered.
func (l *mysqlValidateForEditorListener) EnterUtilityStatement(ctx *parser.UtilityStatementContext) {
	if ctx.ExplainStatement() == nil {
		l.validate = false
	}
}

// EnterExplainableStatement is called when production explainableStatement is entered.
func (l *mysqlValidateForEditorListener) EnterExplainableStatement(ctx *parser.ExplainableStatementContext) {
	if ctx.DeleteStatement() != nil || ctx.UpdateStatement() != nil || ctx.InsertStatement() != nil || ctx.ReplaceStatement() != nil {
		l.validate = false
	}
}

func extractMySQLChangedResources(currentDatabase string, statement string) ([]SchemaResource, error) {
	treeList, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	l := &mysqlChangedResourceExtractListener{
		currentDatabase: currentDatabase,
		resourceMap:     make(map[string]SchemaResource),
	}

	var result []SchemaResource
	for _, tree := range treeList {
		if tree.Tree == nil {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(l, tree.Tree)
	}

	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result, nil
}

type mysqlChangedResourceExtractListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	resourceMap     map[string]SchemaResource
}

// EnterCreateTable is called when production createTable is entered.
func (l *mysqlChangedResourceExtractListener) EnterCreateTable(ctx *parser.CreateTableContext) {
	resource := SchemaResource{
		Database: l.currentDatabase,
	}
	db, table := NormalizeMySQLTableName(ctx.TableName())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

// EnterDropTable is called when production dropTable is entered.
func (l *mysqlChangedResourceExtractListener) EnterDropTable(ctx *parser.DropTableContext) {
	for _, table := range ctx.TableRefList().AllTableRef() {
		resource := SchemaResource{
			Database: l.currentDatabase,
		}
		db, table := NormalizeMySQLTableRef(table)
		if db != "" {
			resource.Database = db
		}
		resource.Table = table
		l.resourceMap[resource.String()] = resource
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (l *mysqlChangedResourceExtractListener) EnterAlterTable(ctx *parser.AlterTableContext) {
	resource := SchemaResource{
		Database: l.currentDatabase,
	}
	db, table := NormalizeMySQLTableRef(ctx.TableRef())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (l *mysqlChangedResourceExtractListener) EnterRenameTableStatement(ctx *parser.RenameTableStatementContext) {
	for _, pair := range ctx.AllRenamePair() {
		{
			resource := SchemaResource{
				Database: l.currentDatabase,
			}
			db, table := NormalizeMySQLTableRef(pair.TableRef())
			if db != "" {
				resource.Database = db
			}
			resource.Table = table
			l.resourceMap[resource.String()] = resource
		}
		{
			resource := SchemaResource{
				Database: l.currentDatabase,
			}
			db, table := NormalizeMySQLTableName(pair.TableName())
			if db != "" {
				resource.Database = db
			}
			resource.Table = table
			l.resourceMap[resource.String()] = resource
		}
	}
}

func extractMySQLResourceList(currentDatabase string, statement string) ([]SchemaResource, error) {
	treeList, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	l := &mysqlResourceExtractListener{
		currentDatabase: currentDatabase,
		resourceMap:     make(map[string]SchemaResource),
	}

	var result []SchemaResource
	for _, tree := range treeList {
		if tree == nil {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(l, tree.Tree)
	}
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type mysqlResourceExtractListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	resourceMap     map[string]SchemaResource
}

// EnterTableRef is called when production tableRef is entered.
func (l *mysqlResourceExtractListener) EnterTableRef(ctx *parser.TableRefContext) {
	resource := SchemaResource{Database: l.currentDatabase}
	if ctx.DotIdentifier() != nil {
		resource.Table = NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	db, table := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

// NormalizeMySQLTableName normalizes the given table name.
func NormalizeMySQLTableName(ctx parser.ITableNameContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeMySQLTableRef normalizes the given table reference.
func NormalizeMySQLTableRef(ctx parser.ITableRefContext) (string, string) {
	if ctx.QualifiedIdentifier() != nil {
		return normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	}
	if ctx.DotIdentifier() != nil {
		return "", NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	return "", ""
}

// NormalizeMySQLColumnName normalizes the given column name.
func NormalizeMySQLColumnName(ctx parser.IColumnNameContext) (string, string, string) {
	if ctx.Identifier() != nil {
		return "", "", NormalizeMySQLIdentifier(ctx.Identifier())
	}
	return NormalizeMySQLFieldIdentifier(ctx.FieldIdentifier())
}

// NormalizeMySQLFieldIdentifier normalizes the given field identifier.
func NormalizeMySQLFieldIdentifier(ctx parser.IFieldIdentifierContext) (string, string, string) {
	list := []string{}
	if ctx.QualifiedIdentifier() != nil {
		id1, id2 := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
		list = append(list, id1, id2)
	}

	if ctx.DotIdentifier() != nil {
		list = append(list, NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier()))
	}

	for len(list) < 3 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1], list[2]
}

func normalizeMySQLQualifiedIdentifier(qualifiedIdentifier parser.IQualifiedIdentifierContext) (string, string) {
	list := []string{NormalizeMySQLIdentifier(qualifiedIdentifier.Identifier())}
	if qualifiedIdentifier.DotIdentifier() != nil {
		list = append(list, NormalizeMySQLIdentifier(qualifiedIdentifier.DotIdentifier().Identifier()))
	}

	if len(list) == 1 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1]
}

// NormalizeMySQLIdentifier normalizes the given identifier.
func NormalizeMySQLIdentifier(identifier parser.IIdentifierContext) string {
	if identifier.PureIdentifier() != nil {
		if identifier.PureIdentifier().IDENTIFIER() != nil {
			return identifier.PureIdentifier().IDENTIFIER().GetText()
		}
		// For back tick quoted identifier, we need to remove the back tick.
		text := identifier.PureIdentifier().BACK_TICK_QUOTED_ID().GetText()
		return text[1 : len(text)-1]
	}
	return identifier.GetText()
}

// NormalizeMySQLSelectAlias normalizes the given select alias.
func NormalizeMySQLSelectAlias(selectAlias parser.ISelectAliasContext) string {
	if selectAlias.Identifier() != nil {
		return NormalizeMySQLIdentifier(selectAlias.Identifier())
	}
	textString := selectAlias.TextStringLiteral().GetText()
	return textString[1 : len(textString)-1]
}

// NormalizeMySQLIdentifierList normalizes the given identifier list.
func NormalizeMySQLIdentifierList(ctx parser.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, NormalizeMySQLIdentifier(identifier))
	}
	return result
}

// IsMySQLAffectedRowsStatement returns true if the given statement is an affected rows statement.
func IsMySQLAffectedRowsStatement(statement string) bool {
	lexer := parser.NewMySQLLexer(antlr.NewInputStream(statement))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	tokens := stream.GetAllTokens()

	for _, token := range tokens {
		if token.GetChannel() == antlr.TokenDefaultChannel {
			switch token.GetTokenType() {
			case parser.MySQLParserDELETE_SYMBOL, parser.MySQLParserINSERT_SYMBOL, parser.MySQLParserREPLACE_SYMBOL, parser.MySQLParserUPDATE_SYMBOL:
				return true
			default:
				return false
			}
		}
	}

	return false
}
