package base

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ExplainFormat is the plan output an explain request asks for. The API's own
// format enum maps onto it where the request enters, so the parsers stay free of
// the API protos.
type ExplainFormat int

const (
	// ExplainFormatDefault is whatever plan the engine returns when the caller
	// asks for no particular format.
	ExplainFormatDefault ExplainFormat = iota
	ExplainFormatText
	ExplainFormatJSON
	ExplainFormatXML
	ExplainFormatYAML
)

// ExplainStatementFunc returns the statement that plans the single statement
// statement. See ExplainStatement for the contract every engine implements.
type ExplainStatementFunc func(statement string, format ExplainFormat) (string, error)

var (
	explainMux        sync.Mutex
	explainStatements = make(map[storepb.Engine]ExplainStatementFunc)
)

// RegisterExplainStatementFunc registers how engine builds the statement that
// plans another statement. Only an engine that plans by prefixing EXPLAIN to the
// statement registers here; see ExplainStatement.
func RegisterExplainStatementFunc(engine storepb.Engine, f ExplainStatementFunc) {
	explainMux.Lock()
	defer explainMux.Unlock()
	if _, dup := explainStatements[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	explainStatements[engine] = f
}

// ExplainStatement returns the statement that produces a query plan for
// statement, which must be a single statement. It is the single source of that
// rewrite: a driver calls it to build the statement it runs, and the Query
// handler calls it to validate the exact statement that will run (an EXPLAIN
// ANALYZE of a write is not read-only and is refused before execution).
//
// The explain request owns the EXPLAIN clause. A statement that already asks for
// a plan is rebuilt around the statement it plans, so the request's format
// replaces the caller's own options and no request ever plans a plan. The one
// refusal is a caller's EXPLAIN ANALYZE: see ErrExplainAnalyzeExecutes. Where the
// engine's parser cannot say which statement is planned, the statement is
// returned unchanged rather than wrapped in a second EXPLAIN.
//
// It fails for an engine that does not plan by prefixing EXPLAIN, and so does not
// execute the statement it plans — Oracle (EXPLAIN PLAN FOR), SQL Server (SET
// SHOWPLAN), Spanner/BigQuery (plan APIs) — and for an engine with no EXPLAIN at
// all. Those drivers produce a plan on their own and never ask; a caller that
// cannot tell which kind of engine it holds asks HasExplainStatement first.
func ExplainStatement(engine storepb.Engine, statement string, format ExplainFormat) (string, error) {
	f, ok := explainStatements[engine]
	if !ok {
		return "", errors.Errorf("%s does not plan a statement by prefixing EXPLAIN", engine)
	}
	return f(statement, format)
}

// HasExplainStatement reports whether engine plans a statement by prefixing
// EXPLAIN to it, which is what makes the statement that runs something other than
// the one the caller holds.
//
// Lock-free like every other lookup here, and for the same reason: the functions
// are registered from package init functions, before any request runs.
func HasExplainStatement(engine storepb.Engine) bool {
	_, ok := explainStatements[engine]
	return ok
}

// ErrExplainAnalyzeExecutes is what an engine returns for a caller's own EXPLAIN
// ANALYZE. That statement plans by executing what it plans, which an explain
// request never does, and planning it without the ANALYZE would quietly answer a
// question the caller did not ask.
var ErrExplainAnalyzeExecutes = errors.New("EXPLAIN ANALYZE executes the statement it plans, which an explain request never does: run it as a query, or drop ANALYZE to plan the statement")

// StartsWithExplain reports whether the first keyword of statement is EXPLAIN,
// ignoring leading comments and whitespace. It stands in for an AST check on an
// engine whose parser cannot answer, which is enough to keep such an engine from
// planning a plan.
func StartsWithExplain(statement string) bool {
	s := statement
	for {
		s = strings.TrimLeft(s, " \t\r\n\v\f")
		switch {
		case strings.HasPrefix(s, "--"):
			_, rest, found := strings.Cut(s, "\n")
			if !found {
				return false
			}
			s = rest
		case strings.HasPrefix(s, "/*"):
			_, rest, found := strings.Cut(s[2:], "*/")
			if !found {
				return false
			}
			s = rest
		default:
			const explain = "EXPLAIN"
			if len(s) < len(explain) || !strings.EqualFold(s[:len(explain)], explain) {
				return false
			}
			// The keyword ends the statement or is followed by something that
			// cannot continue an identifier, so EXPLAINED is not an EXPLAIN.
			if len(s) == len(explain) {
				return true
			}
			c := s[len(explain)]
			return c != '_' && c != '$' && (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c < 0x80
		}
	}
}
