package base

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SingleSQL is a separate SQL split from multi-SQL.
type SingleSQL struct {
	Text string
	// BaseLine is the line number of the first line of the SQL in the original SQL.
	// HINT: ZERO based.
	BaseLine int
	// The inclusive start position of the SQL starts in the original SQL.
	Start *storepb.Position
	// The inclusive end position of the SQL ends in the original SQL.
	End *storepb.Position
	// The sql is empty, such as `/* comments */;` or just `;`.
	Empty bool

	// ByteOffsetStart is the start position of the sql.
	// This field may not be present for every engine.
	// ByteOffsetStart is intended for sql execution log display. It may not represent the actual sql that is sent to the database.
	ByteOffsetStart int
	// ByteOffsetEnd is the end position of the sql.
	// This field may not be present for every engine.
	// ByteOffsetEnd is intended for sql execution log display. It may not represent the actual sql that is sent to the database.
	ByteOffsetEnd int
}

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
	BaseLine  int
	Err       *SyntaxError
	Statement string
}

// SyntaxError returns the errors.
func (l *ParseErrorListener) SyntaxError(_ antlr.Recognizer, token any, line, column int, message string, _ antlr.RecognitionException) {
	if l.Err != nil {
		return
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
		l.Err = &SyntaxError{
			Position: &storepb.Position{
				Line:   int32(line + l.BaseLine),
				Column: int32(column),
			},
			RawMessage: message,
			Message:    fmt.Sprintf("Syntax error at line %d:%d \n%s", line+l.BaseLine, column, errMessage),
		}
		return
	}

	p := common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
		Line:   int32(line),
		Column: int32(column),
	}, l.Statement)
	p.Line += int32(l.BaseLine)
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

func FilterEmptySQL(list []SingleSQL) []SingleSQL {
	var result []SingleSQL
	for _, sql := range list {
		if !sql.Empty {
			result = append(result, sql)
		}
	}
	return result
}

func FilterEmptySQLWithIndexes(list []SingleSQL) ([]SingleSQL, []int32) {
	var result []SingleSQL
	var originalIndex []int32
	for i, sql := range list {
		if !sql.Empty {
			result = append(result, sql)
			originalIndex = append(originalIndex, int32(i))
		}
	}
	return result, originalIndex
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
