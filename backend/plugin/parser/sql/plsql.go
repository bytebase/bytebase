package parser

import (
	"fmt"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	parser "github.com/bytebase/plsql-parser"
)

// SyntaxError is a syntax error.
type SyntaxError struct {
	Line    int
	Column  int
	Message string
}

// Error returns the error message.
func (e *SyntaxError) Error() string {
	return e.Message
}

// PLSQLErrorListener is a custom error listener for PLSQL parser.
type PLSQLErrorListener struct {
	err *SyntaxError
}

// NewPLSQLErrorListener creates a new PLSQLErrorListener.
func NewPLSQLErrorListener() *PLSQLErrorListener {
	return &PLSQLErrorListener{}
}

// SyntaxError returns the errors.
func (l *PLSQLErrorListener) SyntaxError(_ antlr.Recognizer, _ any, line, column int, msg string, _ antlr.RecognitionException) {
	if len(msg) > 1024 {
		msg = msg[:1024]
	}
	if l.err == nil {
		l.err = &SyntaxError{
			Line:    line,
			Column:  column,
			Message: fmt.Sprintf("line %d:%d %s", line, column, msg),
		}
	} else {
		l.err.Message = fmt.Sprintf("%s \nline %d:%d %s", l.err.Message, line, column, msg)
	}
}

// ReportAmbiguity reports an ambiguity.
func (*PLSQLErrorListener) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportAmbiguity(recognizer, dfa, startIndex, stopIndex, exact, ambigAlts, configs)
}

// ReportAttemptingFullContext reports an attempting full context.
func (*PLSQLErrorListener) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportAttemptingFullContext(recognizer, dfa, startIndex, stopIndex, conflictingAlts, configs)
}

// ReportContextSensitivity reports a context sensitivity.
func (*PLSQLErrorListener) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportContextSensitivity(recognizer, dfa, startIndex, stopIndex, prediction, configs)
}

// ParsePLSQL parses the given PLSQL.
func ParsePLSQL(sql string) (antlr.Tree, error) {
	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(sql))
	steam := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewPlSqlParser(steam)
	p.SetVersion12(true)

	lexerErrorListener := &PLSQLErrorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &PLSQLErrorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true
	tree := p.Sql_script()

	if lexerErrorListener.err != nil {
		return nil, lexerErrorListener.err
	}

	if parserErrorListener.err != nil {
		return nil, parserErrorListener.err
	}

	return tree, nil
}

// PLSQLEquivalentType returns true if the given type is equivalent to the given text.
func PLSQLEquivalentType(tp parser.IDatatypeContext, text string) (bool, error) {
	tree, err := ParsePLSQL(fmt.Sprintf(`CREATE TABLE t(a %s);`, text))
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
		l.equivalent = equivalentType(l.tp, ctx.Datatype())
	}
}

func equivalentType(lType parser.IDatatypeContext, rType parser.IDatatypeContext) bool {
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

// IsOracleKeyword returns true if the given text is an Oracle keyword.
func IsOracleKeyword(text string) bool {
	if len(text) == 0 {
		return false
	}

	lexer := parser.NewPlSqlLexer(antlr.NewInputStream(text))
	for _, keyword := range lexer.GetLiteralNames() {
		if strings.EqualFold(strings.Trim(keyword, "'"), text) {
			return true
		}
	}
	return false
}
