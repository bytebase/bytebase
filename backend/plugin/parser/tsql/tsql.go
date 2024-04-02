package tsql

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type ParseResult struct {
	Tree   antlr.Tree
	Tokens *antlr.CommonTokenStream
}

// ParseTSQL parses the given SQL statement by using antlr4. Returns the AST and token stream if no error.
func ParseTSQL(statement string) (*ParseResult, error) {
	statement = strings.TrimRight(statement, " \t\n\r\f;") + "\n;"
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewTSqlLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewTSqlParser(stream)

	// Remove default error listener and add our own error listener.
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{}
	lexer.AddErrorListener(lexerErrorListener)

	p.RemoveErrorListeners()
	parserErrorListener := &base.ParseErrorListener{}
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Tsql_file()

	if lexerErrorListener.Err != nil {
		return nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, parserErrorListener.Err
	}

	result := &ParseResult{
		Tree:   tree,
		Tokens: stream,
	}

	return result, nil
}

// NormalizeTSQLTableName returns the normalized table name.
func NormalizeTSQLTableName(ctx parser.ITable_nameContext, fallbackDatabaseName, fallbackSchemaName string, _ bool) string {
	database := fallbackDatabaseName
	schema := fallbackSchemaName
	table := ""
	if d := ctx.GetDatabase(); d != nil {
		if id, _ := NormalizeTSQLIdentifier(d); id != "" {
			database = id
		}
	}
	if s := ctx.GetSchema(); s != nil {
		if id, _ := NormalizeTSQLIdentifier(s); id != "" {
			schema = id
		}
	}
	if t := ctx.GetTable(); t != nil {
		if id, _ := NormalizeTSQLIdentifier(t); id != "" {
			table = id
		}
	}
	tableNameParts := []string{}
	if database != "" {
		tableNameParts = append(tableNameParts, database)
	}
	if schema != "" {
		tableNameParts = append(tableNameParts, schema)
	}
	if table != "" {
		tableNameParts = append(tableNameParts, table)
	}
	return strings.Join(tableNameParts, ".")
}

// NormalizeTSQLIdentifier returns the normalized identifier.
// https://learn.microsoft.com/zh-cn/sql/relational-databases/databases/database-identifiers?view=sql-server-ver15
// TODO(zp): currently, we returns the lowercase and original of the part, we may need to get the CI/CS from the server/database.
func NormalizeTSQLIdentifier(part parser.IId_Context) (original string, lowercase string) {
	if part == nil {
		return "", ""
	}
	text := part.GetText()
	if text == "" {
		return "", ""
	}
	if text[0] == '[' && text[len(text)-1] == ']' {
		text = text[1 : len(text)-1]
	}

	s := ""
	for _, r := range text {
		s += string(unicode.ToLower(r))
	}
	return text, s
}

// IsTSQLKeyword returns true if the given keyword is a TSQL keywords.
func IsTSQLKeyword(keyword string, caseSensitive bool) bool {
	if !caseSensitive {
		keyword = strings.ToUpper(keyword)
	}
	return tsqlKeywordsMap[keyword]
}

// FlattenExecuteStatementArgExecuteStatementArgUnnamed returns the flattened unnamed execute statement arg.
func FlattenExecuteStatementArgExecuteStatementArgUnnamed(ctx parser.IExecute_statement_argContext) []parser.IExecute_statement_arg_unnamedContext {
	var queue []parser.IExecute_statement_arg_unnamedContext
	ele := ctx
	for {
		if ele.Execute_statement_arg_unnamed() == nil {
			break
		}
		queue = append(queue, ele.Execute_statement_arg_unnamed())
		if len(ele.AllExecute_statement_arg()) != 1 {
			break
		}
		ele = ele.AllExecute_statement_arg()[0]
	}
	return queue
}
