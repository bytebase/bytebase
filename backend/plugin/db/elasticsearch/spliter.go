package elasticsearch

import (
	"bufio"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

func splitElasticsearchStatements(statementsStr string) ([]*statement, error) {
	var stmts []*statement
	sm := &stateMachine{}

	for idx, c := range statementsStr {
		stmt, err := sm.transfer(c)
		if err == nil {
			stmts = append(stmts, stmt)
			continue
		}
		if !strings.Contains(err.Error(), "incomplete") {
			return nil, errors.Wrap(err, "failed to parse statements")
		}
		if idx == len(statementsStr)-1 && (sm.state == statusRoute || sm.state == statusQueryBody) {
			if stmt, err = sm.generateStatement(); err != nil {
				return nil, err
			}
			stmts = append(stmts, stmt)
		}
	}

	return stmts, nil
}

type state int

// States for FSM.
const (
	statusInit state = iota
	statusMethod
	statusRoute
	statusQueryBody

	statusTerminate
)

// Supported HTTP methods for Elasticsearch API.
var (
	supportedHTTPMethods = map[string]bool{
		http.MethodGet:    true,
		http.MethodPost:   true,
		http.MethodPut:    true,
		http.MethodHead:   true,
		http.MethodDelete: true,
	}
)

type statement struct {
	method    []byte
	route     []byte
	queryBody []byte
}

type stateMachine struct {
	state          state
	methodBuf      []byte
	routeBuf       []byte
	queryBodyBuf   []byte
	numLeftBraces  int
	numRightBraces int

	stmt *statement
	err  error
}

func (sm *stateMachine) clear() {
	sm.state = statusInit
	sm.methodBuf = nil
	sm.routeBuf = nil
	sm.queryBodyBuf = nil
	sm.numLeftBraces = 0
	sm.numRightBraces = 0
}

// Perform logic checks and generate a statement.
func (sm *stateMachine) generateStatement() (*statement, error) {
	if !(sm.state == statusRoute || sm.state == statusQueryBody) {
		return nil, errors.Errorf("expect Route or QueryBody, get %v", sm.state)
	}

	// Case insensitive, similar to Kibana's approach.
	upperCaseMethod := strings.ToUpper(string(sm.methodBuf))
	if !supportedHTTPMethods[upperCaseMethod] {
		return nil, errors.Errorf("unsupported method type %q", string(sm.methodBuf))
	}
	if len(sm.routeBuf) == 0 {
		return nil, errors.New("required route is missing")
	}
	// It's ok for routes without the leading '/' in the editor.
	if sm.routeBuf[0] != '/' {
		sm.routeBuf = append([]byte{'/'}, sm.routeBuf...)
	}
	if sm.queryBodyBuf != nil {
		if sm.numLeftBraces != sm.numRightBraces {
			return nil, errors.New("unclosed brace")
		}
		// Elasticsearch Bulk APIs need a '\n' as the end character for the query body.
		if strings.Contains(string(sm.routeBuf), "_bulk") && sm.queryBodyBuf[len(sm.queryBodyBuf)-1] != '\n' {
			sm.queryBodyBuf = append(sm.queryBodyBuf, '\n')
		}
	}
	return &statement{
		method:    []byte(upperCaseMethod),
		route:     sm.routeBuf,
		queryBody: sm.queryBodyBuf,
	}, nil
}

func (sm *stateMachine) needMore() bool {
	if sm.err != nil {
		return false
	}
	if sm.state == statusTerminate {
		return false
	}
	return true
}

func (sm *stateMachine) error() error {
	return sm.err
}

func (sm *stateMachine) transfer(reader *bufio.Reader) (*statement, error) {
	c, _, err := reader.ReadRune()
	if err == io.EOF {
		stmt, err := sm.generateStatement()
		if err != nil {
			sm.err = err
			return nil, sm.err
		}
		sm.stmt = stmt
		sm.state = statusTerminate
	} else if err != nil {
		sm.err = err
		return nil, sm.err
	}

	switch sm.state {
	case statusInit:
		if isASCIIAlpha(c) {
			sm.state = statusMethod
			sm.methodBuf = append(sm.methodBuf, string(c)...)
		} else if c != '\r' && c != '\n' && c != ' ' {
			sm.err = errors.Errorf("invalid character %q for method", c)
			return nil, sm.err
		}

	case statusMethod:
		if c == ' ' {
			sm.state = statusRoute
		} else if isASCIIAlpha(c) {
			sm.methodBuf = append(sm.methodBuf, string(c)...)
		} else {
			sm.err = errors.Errorf("invalid character %q for method", c)
			return nil, sm.err
		}

	case statusRoute:
		if c == '\n' {
			if sm.routeBuf == nil {
				sm.err = errors.New("required route is missing")
				return nil, sm.err
			}
			sm.state = statusQueryBody
			// Ignore CR characters produced by line breaks on Windows.
		} else if c != '\r' && c != ' ' {
			sm.routeBuf = append(sm.routeBuf, string(c)...)
		}

	case statusQueryBody:
		// Return a valid statement when:
		// 1. An alphabetic character is encountered, which represents the start of a method in the next statement.
		// 2. A newline character is encountered and there are no left braces.
		if (isASCIIAlpha(c) && (sm.numLeftBraces == sm.numRightBraces)) || (c == '\n' && sm.numLeftBraces == 0) {
			stmt, err := sm.generateStatement()
			if err != nil {
				sm.err = err
				return nil, sm.err
			}
			sm.stmt = stmt
			sm.state = statusTerminate

			sm.clear()
			if isASCIIAlpha(c) {
				sm.methodBuf = append(sm.methodBuf, string(c)...)
				sm.state = statusMethod
			}
			return stmt, nil
		}

		// Ignore any characters other than '\n', '{' and '}' outside the braces.
		if c == '\n' || c == '{' || c == '}' || sm.numLeftBraces > sm.numRightBraces {
			sm.queryBodyBuf = append(sm.queryBodyBuf, string(c)...)
		}

		if c == '{' {
			sm.numLeftBraces++
		} else if c == '}' {
			sm.numRightBraces++
			if sm.numLeftBraces < sm.numRightBraces {
				sm.err = errors.New("the curly braces '{}' are mismatched")
				return nil, sm.err
			}
		}

	default:
	}
	return nil, errors.New("incomplete")
}

func isASCIIAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
