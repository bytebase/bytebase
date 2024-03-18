package hive

import (
	"errors"
	"unicode"
)

// TODO(tommy): need more tests
// enumeration for statement parsing state.
type StatementStatus int32

var (
	StatementEnd   StatementStatus
	StatementBegin StatementStatus = 1
	CommentStart   StatementStatus = 2
	Comment        StatementStatus = 3
	CommentCR      StatementStatus = 4
	Error          StatementStatus = 5
)

func splitHiveStatements(statementsStr string) ([]string, error) {
	var (
		statementStartIndex int
		statements          []string
		state               StatementStatus
		isStatementStarted  = false
	)

	for currIndex, c := range statementsStr {
		transferState(&state, int32(c))
		switch state {
		case Error:
			return nil, errors.New("syntax error")
		case StatementBegin:
			if !isStatementStarted {
				statementStartIndex = currIndex
				isStatementStarted = true
			}
			if currIndex == len(statementsStr)-1 {
				statements = append(statements, statementsStr[statementStartIndex:currIndex-1])
			}
		case StatementEnd:
			isStatementStarted = false
			statements = append(statements, statementsStr[statementStartIndex:currIndex-1])
		}
	}

	return statements, nil
}

func transferState(state *StatementStatus, input int32) {
	switch *state {
	case StatementEnd:
		if isStatementChar(input) {
			*state = StatementBegin
		} else if input == '-' {
			*state = CommentStart
		} else if input != ';' {
			*state = Error
		}

	case StatementBegin:
		if input == ';' {
			*state = StatementEnd
		} else if input == '-' {
			*state = CommentStart
		} else if !isStatementChar(input) {
			*state = Error
		}

	case CommentStart:
		if input == '-' {
			*state = Comment
		} else {
			*state = Error
		}

	case Comment:
		if input == '\r' {
			*state = CommentCR
		} else if input == '\n' {
			*state = StatementEnd
		}

	case CommentCR:
		if input != '\n' {
			*state = Error
			return
		}
		*state = StatementEnd

	default:
	}
}

// this function will test whether a certain char can occur in Hive's statement.
func isStatementChar(input int32) bool {
	if unicode.IsLetter(input) {
		return true
	}

	statementChars := []int32{'\r', '\n', '\t', ',', '*', '\'', ' '}
	for _, c := range statementChars {
		if input == int32(c) {
			return true
		}
	}

	return false
}
