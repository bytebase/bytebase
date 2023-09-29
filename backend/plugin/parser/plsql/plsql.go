package plsql

import (
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
