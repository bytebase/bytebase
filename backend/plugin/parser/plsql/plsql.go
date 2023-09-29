package plsql

import (
	"io"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ParsePLSQL parses the given PLSQL.
func ParsePLSQL(sql string) (antlr.Tree, *antlr.CommonTokenStream, error) {
	sql = addSemicolonIfNeeded(sql)
	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewPlSqlParser(stream)
	p.SetVersion12(true)

	lexerErrorListener := &base.ParseErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true
	tree := p.Sql_script()

	if lexerErrorListener.Err != nil {
		return nil, nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, nil, parserErrorListener.Err
	}

	return tree, stream, nil
}

func addSemicolonIfNeeded(sql string) string {
	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(sql))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	stream.Fill()
	tokens := stream.GetAllTokens()
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].GetChannel() != antlr.TokenDefaultChannel || tokens[i].GetTokenType() == parser.PlSqlParserEOF {
			continue
		}

		// The last default channel token is a semicolon.
		if tokens[i].GetTokenType() == parser.PlSqlParserSEMICOLON {
			return sql
		}

		return stream.GetTextFromInterval(antlr.NewInterval(0, tokens[i].GetTokenIndex())) + ";"
	}
	return sql
}

// IsOracleKeyword returns true if the given text is an Oracle keyword.
func IsOracleKeyword(text string) bool {
	if len(text) == 0 {
		return false
	}

	return oracleKeywords[strings.ToUpper(text)] || oracleReservedWords[strings.ToUpper(text)]
}

// NormalizeConstraintName returns the normalized constraint name from the given context.
func NormalizeConstraintName(constraintName parser.IConstraint_nameContext) (string, string) {
	if constraintName == nil {
		return "", ""
	}

	if constraintName.Id_expression(0) != nil {
		return NormalizeIdentifierContext(constraintName.Identifier()),
			NormalizeIDExpression(constraintName.Id_expression(0))
	}

	return "", NormalizeIdentifierContext(constraintName.Identifier())
}

// NormalizeIdentifierContext returns the normalized identifier from the given context.
func NormalizeIdentifierContext(identifier parser.IIdentifierContext) string {
	if identifier == nil {
		return ""
	}

	return NormalizeIDExpression(identifier.Id_expression())
}

// NormalizeIDExpression returns the normalized identifier from the given context.
func NormalizeIDExpression(idExpression parser.IId_expressionContext) string {
	if idExpression == nil {
		return ""
	}

	regularID := idExpression.Regular_id()
	if regularID != nil {
		return strings.ToUpper(regularID.GetText())
	}

	delimitedID := idExpression.DELIMITED_ID()
	if delimitedID != nil {
		return strings.Trim(delimitedID.GetText(), "\"")
	}

	return ""
}

// NormalizeIndexName returns the normalized index name from the given context.
func NormalizeIndexName(indexName parser.IIndex_nameContext) (string, string) {
	if indexName == nil {
		return "", ""
	}

	if indexName.Id_expression() != nil {
		return NormalizeIdentifierContext(indexName.Identifier()),
			NormalizeIDExpression(indexName.Id_expression())
	}

	return "", NormalizeIdentifierContext(indexName.Identifier())
}

// SplitMultiSQLStream splits MySQL multiSQL to stream.
// Note that the reader is read completely into memory and so it must actually
// have a stopping point - you cannot pass in a reader on an open-ended source such
// as a socket for instance.
func SplitMultiSQLStream(src io.Reader, f func(string) error) ([]base.SingleSQL, error) {
	text := antlr.NewIoStream(src).String()
	sqls, err := SplitPLSQL(text)
	if err != nil {
		return nil, err
	}
	for _, sql := range sqls {
		if f != nil {
			if err := f(sql.Text); err != nil {
				return nil, err
			}
		}
	}
	return sqls, nil
}

// SplitPLSQL splits the given SQL statement into multiple SQL statements.
func SplitPLSQL(statement string) ([]base.SingleSQL, error) {
	tree, tokens, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.SingleSQL
	for _, item := range tree.GetChildren() {
		if stmt, ok := item.(parser.IUnit_statementContext); ok {
			stopIndex := stmt.GetStop().GetTokenIndex()
			if stmt.GetStop().GetTokenType() == parser.PlSqlParserSEMICOLON {
				stopIndex--
			}
			lastToken := tokens.Get(stopIndex)
			text := tokens.GetTextFromTokens(stmt.GetStart(), lastToken)
			text = strings.TrimRight(text, " \n\t;")

			result = append(result, base.SingleSQL{
				Text:     text,
				LastLine: lastToken.GetLine(),
				Empty:    false,
			})
		}
	}
	return result, nil
}
