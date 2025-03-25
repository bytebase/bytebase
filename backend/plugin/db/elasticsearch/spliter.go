package elasticsearch

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
)

func splitElasticsearchStatements(statementsStr string) ([]*statement, error) {
	var stmts []*statement
	sm := &stateMachine{}

	reader := bufio.NewReader(strings.NewReader(statementsStr))

	for {
		sm.reset()
		for sm.needMore() {
			sm.consume(reader)
		}
		if sm.error() != nil {
			return nil, sm.error()
		}
		if sm.statement() != nil {
			stmts = append(stmts, sm.statement())
		}

		if sm.eof {
			break
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
	eof  bool
}

func (sm *stateMachine) reset() {
	sm.state = statusInit
	sm.methodBuf = nil
	sm.routeBuf = nil
	sm.queryBodyBuf = nil
	sm.numLeftBraces = 0
	sm.numRightBraces = 0

	sm.stmt = nil
	sm.err = nil
	sm.eof = false
}

// Perform logic checks and generate a statement.
func (sm *stateMachine) generateStatement() (*statement, error) {
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

func (sm *stateMachine) statement() *statement {
	return sm.stmt
}

func (sm *stateMachine) consume(reader *bufio.Reader) {
	c, _, err := reader.ReadRune()
	if err == io.EOF {
		sm.eof = true
		// maybe the previous run left us some empty chars to read.
		if sm.state == statusRoute || sm.state == statusQueryBody {
			stmt, err := sm.generateStatement()
			if err != nil {
				sm.err = err
				return
			}
			sm.stmt = stmt
		}
		sm.state = statusTerminate
		return
	} else if err != nil {
		sm.err = err
		return
	}

	switch sm.state {
	case statusInit:
		if isASCIIAlpha(c) {
			sm.state = statusMethod
			sm.methodBuf = append(sm.methodBuf, string(c)...)
		} else if c != '\r' && c != '\n' && c != ' ' {
			sm.err = errors.Errorf("invalid character %q for method", c)
			return
		}

	case statusMethod:
		if c == ' ' {
			sm.state = statusRoute
		} else if isASCIIAlpha(c) {
			sm.methodBuf = append(sm.methodBuf, string(c)...)
		} else {
			sm.err = errors.Errorf("invalid character %q for method", c)
			return
		}

	case statusRoute:
		if c == '\n' {
			if sm.routeBuf == nil {
				sm.err = errors.New("required route is missing")
				return
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
			if isASCIIAlpha(c) {
				if err := reader.UnreadRune(); err != nil {
					slog.Warn("failed to unread rune", log.BBError(err))
				}
			}
			stmt, err := sm.generateStatement()
			if err != nil {
				sm.err = err
				return
			}
			sm.stmt = stmt
			sm.state = statusTerminate
			return
		}

		// Ignore any characters other than '\n', '{' and '}' outside the braces.
		if c == '\n' || c == '{' || c == '}' || sm.numLeftBraces > sm.numRightBraces {
			sm.queryBodyBuf = append(sm.queryBodyBuf, string(c)...)
		}

		switch c {
		case '{':
			sm.numLeftBraces++
		case '}':
			sm.numRightBraces++
			if sm.numLeftBraces < sm.numRightBraces {
				sm.err = errors.New("the curly braces '{}' are mismatched")
				return
			}
		}
	default:
		sm.err = errors.Errorf("unexpected state %v", sm.state)
	}
}

func isASCIIAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
