package parser

import (
	"fmt"

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
