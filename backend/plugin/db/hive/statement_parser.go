package hive

import (
	"errors"
	"unicode"
)

// TODO(tommy): need more tests
// enumeration for statement parsing state.
type StatementStatus int32

var (
	StatementInit  StatementStatus
	StatementEnd   StatementStatus = 1
	StatementBegin StatementStatus = 2
	CommentEnd     StatementStatus = 3
	CommentBegin   StatementStatus = 4
	Comment        StatementStatus = 5
	CommentCR      StatementStatus = 6
	Error          StatementStatus = 7
)

func splitHiveStatements(statementsStr string) ([]string, error) {
	var (
		// statementStartIndex int
		state           = StatementInit
		statements      = []string{}
		statementBuffer = []int32{}
		// isStatementStarted  = false
		// isCommentStarted    = false
	)

	for idx, c := range statementsStr {
		transferState(&state, int32(c))

		switch state {
		case Error:
			return nil, errors.New("syntax error")
		case StatementBegin:
			statementBuffer = append(statementBuffer, int32(c))
			if idx == len(statementsStr)-1 {
				statements = append(statements, string(statementBuffer))
			}
		case StatementEnd:
			if len(statementBuffer) == 0 {
				continue
			}
			statements = append(statements, string(statementBuffer))
			statementBuffer = []int32{}
		default:
		}
	}

	return statements, nil
}

func transferState(state *StatementStatus, input int32) {
	switch *state {
	case StatementInit:
		if isStatementChar(input) {
			*state = StatementBegin
		} else if input == '-' {
			*state = CommentBegin
		}

	case StatementEnd:
		if isStatementChar(input) {
			*state = StatementBegin
		} else if input == '-' {
			*state = CommentBegin
		}

	case StatementBegin:
		if input == ';' {
			*state = StatementEnd
		} else if input == '-' {
			*state = CommentBegin
		} else if !isStatementChar(input) {
			*state = Error
		}

	case CommentBegin:
		if input == '-' {
			*state = Comment
		} else {
			*state = Error
		}

	case Comment:
		if input == '\r' {
			*state = CommentCR
		} else if input == '\n' {
			*state = CommentEnd
		}

	case CommentCR:
		if input != '\n' {
			*state = Error
			return
		}
		*state = StatementInit

	case CommentEnd:
		if isStatementChar(input) {
			*state = StatementBegin
		} else if input == '-' {
			*state = CommentBegin
		}
	default:
	}
}

// this function will test whether a certain char can occur in Hive's statement.
func isStatementChar(input int32) bool {
	if unicode.IsLetter(input) || unicode.IsDigit(input) {
		return true
	}

	statementChars := []int32{'\r', '\n', '\t', ',', '*', '\'', ' ', '_', '(', ')'}
	for _, c := range statementChars {
		if input == int32(c) {
			return true
		}
	}

	return false
}
