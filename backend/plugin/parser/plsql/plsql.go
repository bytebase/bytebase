package plsql

import (
	"fmt"
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

// EquivalentType returns true if the given type is equivalent to the given text.
func EquivalentType(tp parser.IDatatypeContext, text string) (bool, error) {
	tree, _, err := ParsePLSQL(fmt.Sprintf(`CREATE TABLE t(a %s);`, text))
	if err != nil {
		return false, err
	}

	listener := &typeEquivalentListener{tp: tp, equivalent: false}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.equivalent, nil
}

type typeEquivalentListener struct {
	*parser.BasePlSqlParserListener

	tp         parser.IDatatypeContext
	equivalent bool
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *typeEquivalentListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if ctx.Datatype() != nil {
		l.equivalent = equalDataType(l.tp, ctx.Datatype())
	}
}

func equalDataType(lType parser.IDatatypeContext, rType parser.IDatatypeContext) bool {
	if lType == nil || rType == nil {
		return false
	}
	lNative := lType.Native_datatype_element()
	rNative := rType.Native_datatype_element()

	if lNative != nil && rNative != nil {
		switch {
		case lNative.BINARY_INTEGER() != nil:
			return rNative.BINARY_INTEGER() != nil
		case lNative.PLS_INTEGER() != nil:
			return rNative.PLS_INTEGER() != nil
		case lNative.NATURAL() != nil:
			return rNative.NATURAL() != nil
		case lNative.BINARY_FLOAT() != nil:
			return rNative.BINARY_FLOAT() != nil
		case lNative.BINARY_DOUBLE() != nil:
			return rNative.BINARY_DOUBLE() != nil
		case lNative.NATURALN() != nil:
			return rNative.NATURALN() != nil
		case lNative.POSITIVE() != nil:
			return rNative.POSITIVE() != nil
		case lNative.POSITIVEN() != nil:
			return rNative.POSITIVEN() != nil
		case lNative.SIGNTYPE() != nil:
			return rNative.SIGNTYPE() != nil
		case lNative.SIMPLE_INTEGER() != nil:
			return rNative.SIMPLE_INTEGER() != nil
		case lNative.NVARCHAR2() != nil:
			return rNative.NVARCHAR2() != nil
		case lNative.DEC() != nil:
			return rNative.DEC() != nil
		case lNative.INTEGER() != nil:
			return rNative.INTEGER() != nil
		case lNative.INT() != nil:
			return rNative.INT() != nil
		case lNative.NUMERIC() != nil:
			return rNative.NUMERIC() != nil
		case lNative.SMALLINT() != nil:
			return rNative.SMALLINT() != nil
		case lNative.NUMBER() != nil:
			return rNative.NUMBER() != nil
		case lNative.DECIMAL() != nil:
			return rNative.DECIMAL() != nil
		case lNative.DOUBLE() != nil:
			return rNative.DOUBLE() != nil
		case lNative.FLOAT() != nil:
			return rNative.FLOAT() != nil
		case lNative.REAL() != nil:
			return rNative.REAL() != nil
		case lNative.NCHAR() != nil:
			return rNative.NCHAR() != nil
		case lNative.LONG() != nil:
			return rNative.LONG() != nil
		case lNative.CHAR() != nil:
			return rNative.CHAR() != nil
		case lNative.CHARACTER() != nil:
			return rNative.CHARACTER() != nil
		case lNative.VARCHAR2() != nil:
			return rNative.VARCHAR2() != nil
		case lNative.VARCHAR() != nil:
			return rNative.VARCHAR() != nil
		case lNative.STRING() != nil:
			return rNative.STRING() != nil
		case lNative.RAW() != nil:
			return rNative.RAW() != nil
		case lNative.BOOLEAN() != nil:
			return rNative.BOOLEAN() != nil
		case lNative.DATE() != nil:
			return rNative.DATE() != nil
		case lNative.ROWID() != nil:
			return rNative.ROWID() != nil
		case lNative.UROWID() != nil:
			return rNative.UROWID() != nil
		case lNative.YEAR() != nil:
			return rNative.YEAR() != nil
		case lNative.MONTH() != nil:
			return rNative.MONTH() != nil
		case lNative.DAY() != nil:
			return rNative.DAY() != nil
		case lNative.HOUR() != nil:
			return rNative.HOUR() != nil
		case lNative.MINUTE() != nil:
			return rNative.MINUTE() != nil
		case lNative.SECOND() != nil:
			return rNative.SECOND() != nil
		case lNative.TIMEZONE_HOUR() != nil:
			return rNative.TIMEZONE_HOUR() != nil
		case lNative.TIMEZONE_MINUTE() != nil:
			return rNative.TIMEZONE_MINUTE() != nil
		case lNative.TIMEZONE_REGION() != nil:
			return rNative.TIMEZONE_REGION() != nil
		case lNative.TIMEZONE_ABBR() != nil:
			return rNative.TIMEZONE_ABBR() != nil
		case lNative.TIMESTAMP() != nil:
			return rNative.TIMESTAMP() != nil
		case lNative.TIMESTAMP_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil
		case lNative.YMINTERVAL_UNCONSTRAINED() != nil:
			return rNative.YMINTERVAL_UNCONSTRAINED() != nil
		case lNative.DSINTERVAL_UNCONSTRAINED() != nil:
			return rNative.DSINTERVAL_UNCONSTRAINED() != nil
		case lNative.BFILE() != nil:
			return rNative.BFILE() != nil
		case lNative.BLOB() != nil:
			return rNative.BLOB() != nil
		case lNative.CLOB() != nil:
			return rNative.CLOB() != nil
		case lNative.NCLOB() != nil:
			return rNative.NCLOB() != nil
		case lNative.MLSLABEL() != nil:
			return rNative.MLSLABEL() != nil
		default:
			return false
		}
	}

	if lNative != nil || rNative != nil {
		return false
	}

	return lType.GetText() == rType.GetText()
}
