package parser

import (
	"errors"
	"strconv"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	parser "github.com/bytebase/plsql-parser"
)

// PLSQLErrorListener is a custom error listener for PLSQL parser.
type PLSQLErrorListener struct {
	errors []string
}

// NewPLSQLErrorListener creates a new PLSQLErrorListener.
func NewPLSQLErrorListener() *PLSQLErrorListener {
	return &PLSQLErrorListener{}
}

// SyntaxError returns the errors.
func (l *PLSQLErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	if len(msg) > 1024 {
		msg = msg[:1024]
	}
	l.errors = append(l.errors, "line "+strconv.Itoa(line)+":"+strconv.Itoa(column)+" "+msg)
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

	if len(lexerErrorListener.errors) > 0 {
		return nil, errors.New(strings.Join(lexerErrorListener.errors, " \n"))
	}

	if len(parserErrorListener.errors) > 0 {
		return nil, errors.New(strings.Join(parserErrorListener.errors, " \n"))
	}

	return tree, nil
}
