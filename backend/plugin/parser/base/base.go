package base

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SyntaxError is a syntax error.
type SyntaxError struct {
	Position   *storepb.Position
	Message    string
	RawMessage string
}

// Error returns the error message.
func (e *SyntaxError) Error() string {
	return e.Message
}

// ParseErrorListener is a custom error listener for PLSQL parser.
type ParseErrorListener struct {
	// StartPosition is the 1-based position where this statement starts in the original multi-statement input.
	// Used to calculate error positions relative to the original script.
	StartPosition *storepb.Position
	Err           *SyntaxError
	Statement     string
}

// SyntaxError returns the errors.
func (l *ParseErrorListener) SyntaxError(_ antlr.Recognizer, token any, line, column int, message string, _ antlr.RecognitionException) {
	if l.Err != nil {
		return
	}

	// Get 0-based line offset from StartPosition (1-based) for calculations
	lineOffset := int32(0)
	if l.StartPosition != nil {
		lineOffset = l.StartPosition.Line - 1
	}

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
	if l.Statement == "" {
		// ANTLR provides 1-based line and 0-based column
		// Store as 1-based line and 1-based column in Position
		posLine := int32(line) + lineOffset
		posColumn := int32(column + 1)
		l.Err = &SyntaxError{
			Position: &storepb.Position{
				Line:   posLine,
				Column: posColumn,
			},
			RawMessage: message,
			// Display directly (already 1-based)
			Message: fmt.Sprintf("Syntax error at line %d:%d \n%s", posLine, posColumn, errMessage),
		}
		return
	}

	p := common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
		Line:   int32(line),
		Column: int32(column),
	}, l.Statement)
	p.Line += lineOffset
	l.Err = &SyntaxError{
		Position: &storepb.Position{
			Line:   p.Line,
			Column: p.Column,
		},
		RawMessage: message,
		Message:    fmt.Sprintf("Syntax error at line %d:%d \n%s", p.Line, p.Column, errMessage),
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

func GetOffsetLength(total int) int {
	length := 1
	for {
		if total < 10 {
			return length
		}
		total /= 10
		length++
	}
}

// GetLineOffset returns the 0-based line offset from a StartPosition.
// This is useful for converting from the 1-based StartPosition to the 0-based offset
// needed for some calculations.
func GetLineOffset(startPosition *storepb.Position) int {
	if startPosition == nil {
		return 0
	}
	return int(startPosition.Line) - 1
}
