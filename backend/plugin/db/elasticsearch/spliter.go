package elasticsearch

import (
	"strings"

	"github.com/cockroachdb/errors"
)

func splitElasticsearchStatements(statementsStr string) ([]*statement, error) {
	var stats []*statement
	sm := &stateMachine{}

	for idx, c := range statementsStr {
		stat, err := sm.transfer(c)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse statements")
		}
		// Generate a statement when the maximum parsed content length is reached.
		if stat == nil && (idx == len(statementsStr)-1) {
			if stat, err = sm.generateStatement(); err != nil {
				return nil, err
			}
		}
		if stat != nil {
			stats = append(stats, stat)
		}
	}

	return stats, nil
}

type state int

// States for FSM.
const (
	statusInit state = iota
	statusMethod
	statusRoute
	statusQueryBody
)

// Supported HTTP methods for Elasticsearch API.
var (
	supportedHTTPMethods = map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"HEAD":   true,
		"DELETE": true,
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
	if sm.state != statusRoute && sm.state != statusQueryBody {
		return nil, nil
	}
	// Case insensitive, similar to Kibana's approach.
	upperCaseMethod := strings.ToUpper(string(sm.methodBuf))
	if !supportedHTTPMethods[upperCaseMethod] {
		return nil, errors.Newf("unsupported method type %q", string(sm.methodBuf))
	}
	if sm.routeBuf == nil {
		return nil, errors.New("required route is missing")
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

func (sm *stateMachine) transfer(c rune) (*statement, error) {
	switch sm.state {
	case statusInit:
		if isASCIIAlpha(c) {
			sm.state = statusMethod
			sm.methodBuf = append(sm.methodBuf, string(c)...)
		} else if c != '\r' && c != '\n' && c != ' ' {
			return nil, errors.Newf("invalid character %q for method", c)
		}

	case statusMethod:
		if c == ' ' {
			sm.state = statusRoute
		} else if isASCIIAlpha(c) {
			sm.methodBuf = append(sm.methodBuf, string(c)...)
		} else {
			return nil, errors.Newf("invalid character %q for method", c)
		}

	case statusRoute:
		if c == '\n' {
			if sm.routeBuf == nil {
				return nil, errors.New("required route is missing")
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
			stat, err := sm.generateStatement()
			if err != nil {
				return nil, err
			}
			sm.clear()
			if isASCIIAlpha(c) {
				sm.methodBuf = append(sm.methodBuf, string(c)...)
				sm.state = statusMethod
			}
			return stat, nil
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
				return nil, errors.New("the curly braces '{}' are mismatched")
			}
		}

	default:
		return nil, errors.New("invalid state")
	}

	return nil, nil
}

func isASCIIAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
