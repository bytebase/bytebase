package elasticsearch

import (
	"errors"
)

func SplitElasticsearchStatements(statementsStr string) ([]*Statement, error) {
	var sm StateMachine
	var statements []*Statement

	for idx, c := range statementsStr {
		statement := sm.transfer(c)
		if sm.state == StatusError {
			return nil, errors.New("failed to parse statements")
		}
		if statement != nil {
			statements = append(statements, statement)
		} else if idx == len(statementsStr)-1 && isMethodSupported(sm.statement.method) && sm.statement.route != nil {
			tmpStatement := &Statement{
				method: sm.statement.method,
				route:  sm.statement.route,
			}
			if sm.statement.queryString != nil && sm.numLeftBrace == 0 {
				tmpStatement.queryString = sm.statement.queryString
			}
			statements = append(statements, tmpStatement)
		}
	}

	return statements, nil
}

// state for FSM.
const (
	StatusInit = iota
	StatusMethod
	StatusRoute
	StatusQueryBody
	StatusError
)

// supported HTTP methods for elasticsearch API.
var (
	suportedHTTPMethods = []string{"GET", "POST", "PUT", "HEAD", "DELETE"}
)

type Statement struct {
	method      string
	route       []byte
	queryString []byte
}

func (s *Statement) Clear() {
	s.method = ""
	s.queryString = []byte{}
	s.route = []byte{}
}

type StateMachine struct {
	state        int
	statement    Statement
	numLeftBrace int
}

func (sm *StateMachine) transfer(c rune) *Statement {
	switch sm.state {
	case StatusInit:
		if isASCIIAlpha(c) {
			sm.state = StatusMethod
			sm.statement.method += string(c)
		}

	case StatusMethod:
		if isASCIIAlpha(c) {
			sm.statement.method += string(c)
		} else if c == ' ' {
			if !isMethodSupported(sm.statement.method) {
				sm.state = StatusError
			} else {
				sm.state = StatusRoute
			}
		}

	case StatusRoute:
		if c == '\n' {
			if sm.statement.route == nil {
				sm.state = StatusError
			} else {
				sm.state = StatusQueryBody
			}
		} else if c != ' ' {
			sm.statement.route = append(sm.statement.route, string(c)...)
		}

	case StatusQueryBody:
		if isASCIIAlpha(c) && sm.numLeftBrace == 0 {
			statement := &Statement{
				method:      sm.statement.method,
				route:       sm.statement.route,
				queryString: sm.statement.queryString,
			}

			sm.state = StatusInit
			sm.statement.Clear()
			sm.statement.method += string(c)
			return statement
		}
		sm.statement.queryString = append(sm.statement.queryString, string(c)...)
		if c == '{' {
			sm.numLeftBrace++
		} else if c == '}' {
			sm.numLeftBrace--
			if sm.numLeftBrace < 0 {
				sm.state = StatusError
			}
		}
	default:
		sm.state = StatusError
		return nil
	}
	return nil
}

func isASCIIAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isMethodSupported(s string) bool {
	if s == "" {
		return false
	}
	for _, m := range suportedHTTPMethods {
		if s == m {
			return true
		}
	}
	return false
}
