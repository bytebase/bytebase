// Package parser is the parser for SQL statement.
package parser

import (
	"errors"
	"io"
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"
)

// ParseMySQL parses the given SQL statement and returns the AST.
func ParseMySQL(statement string) (antlr.Tree, *antlr.CommonTokenStream, error) {
	return parseInputStream(antlr.NewInputStream(statement))
}

// ParseMySQLStream parses the given SQL stream and returns the AST.
// Note that the reader is read completely into memory and so it must actually
// have a stopping point - you cannot pass in a reader on an open-ended source such
// as a socket for instance.
func ParseMySQLStream(src io.Reader) (antlr.Tree, *antlr.CommonTokenStream, error) {
	return parseInputStream(antlr.NewIoStream(src))
}

func parseInputStream(input *antlr.InputStream) (antlr.Tree, *antlr.CommonTokenStream, error) {
	lexer := parser.NewMySQLLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewMySQLParser(stream)

	lexerErrorListener := &ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Script()

	if lexerErrorListener.err != nil {
		return nil, nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, nil, parserErrorListener.err
	}

	return tree, stream, nil
}

// MySQLValidateForEditor validates the given SQL statement for editor.
func MySQLValidateForEditor(tree antlr.Tree) error {
	l := &mysqlValidateForEditorListener{
		validate: true,
	}

	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return errors.New("Malformed sql execute request, only support SELECT sql statement")
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

func extractMySQLResourceList(currentDatabase string, statement string) ([]SchemaResource, error) {
	tree, _, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	l := &mysqlResourceExtractListener{
		currentDatabase: currentDatabase,
		resourceMap:     make(map[string]SchemaResource),
	}

	var result []SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
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
		resource.Table = normalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	db, table := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}

func normalizeMySQLQualifiedIdentifier(qualifiedIdentifier parser.IQualifiedIdentifierContext) (string, string) {
	list := []string{normalizeMySQLIdentifier(qualifiedIdentifier.Identifier())}
	if qualifiedIdentifier.DotIdentifier() != nil {
		list = append(list, normalizeMySQLIdentifier(qualifiedIdentifier.DotIdentifier().Identifier()))
	}

	if len(list) == 1 {
		list = append([]string{""}, list...)
	}

	return list[0], list[1]
}

func normalizeMySQLIdentifier(identifier parser.IIdentifierContext) string {
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
