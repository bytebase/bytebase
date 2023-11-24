package base

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
)

// SingleSQL is a separate SQL split from multi-SQL.
type SingleSQL struct {
	Text string
	// BaseLine is the line number of the first line of the SQL in the original SQL.
	BaseLine int
	// FirstStatementLine is the line number of the first non-comment and non-blank line of the SQL in the original SQL.
	FirstStatementLine int
	// LastLine is the line number of the last line of the SQL in the original SQL.
	LastLine int
	// The sql is empty, such as `/* comments */;` or just `;`.
	Empty bool
}

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

// ParseErrorListener is a custom error listener for PLSQL parser.
type ParseErrorListener struct {
	Err *SyntaxError
}

// SyntaxError returns the errors.
func (l *ParseErrorListener) SyntaxError(_ antlr.Recognizer, token any, line, column int, _ string, _ antlr.RecognitionException) {
	if l.Err == nil {
		errMessage := ""
		if token, ok := token.(*antlr.CommonToken); ok {
			stream := token.GetInputStream()
			start := token.GetStart() - 40
			if start < 0 {
				start = 0
			}
			stop := token.GetStop()
			if stop >= stream.Size() {
				stop = stream.Size() - 1
			}
			errMessage = fmt.Sprintf("related text: %s", stream.GetTextFromInterval(antlr.NewInterval(start, stop)))
		}
		l.Err = &SyntaxError{
			Line:    line,
			Column:  column,
			Message: fmt.Sprintf("Syntax error at line %d:%d \n%s", line, column, errMessage),
		}
	}
}

// ReportAmbiguity reports an ambiguity.
func (*ParseErrorListener) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportAmbiguity(recognizer, dfa, startIndex, stopIndex, exact, ambigAlts, configs)
}

// ReportAttemptingFullContext reports an attempting full context.
func (*ParseErrorListener) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportAttemptingFullContext(recognizer, dfa, startIndex, stopIndex, conflictingAlts, configs)
}

// ReportContextSensitivity reports a context sensitivity.
func (*ParseErrorListener) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs *antlr.ATNConfigSet) {
	antlr.ConsoleErrorListenerINSTANCE.ReportContextSensitivity(recognizer, dfa, startIndex, stopIndex, prediction, configs)
}
