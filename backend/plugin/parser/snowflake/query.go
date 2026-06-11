package snowflake

import (
	"strings"

	"github.com/bytebase/omni/snowflake/ast"
	"github.com/bytebase/omni/snowflake/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_SNOWFLAKE, validateQuery)
}

// validateQuery validates the SQL statement for the SQL editor, which only
// permits read-only queries.
//
// It returns (allReadOnly, allReturnData, error):
//   - allReadOnly:   every statement can run in read-only mode;
//   - allReturnData: every statement returns data;
//   - error:         a syntax error if a statement is invalid.
//
// This mirrors the legacy ANTLR queryValidateListener exactly. The read-only
// set is {SELECT, SHOW, DESCRIBE, SET, EXPLAIN}; a SET statement is read-only
// but does NOT return data (legacy set hasExecute=true → returnsData=false).
// Every other statement (USE, INSERT/UPDATE/DELETE/MERGE, CALL, COPY, COMMENT,
// TRUNCATE, GRANT/REVOKE, CREATE/ALTER/DROP, ...) is not read-only.
//
// EXPLAIN is special-cased because omni's parser does not yet support it
// (parser.Parse returns an error). The legacy listener accepted any
// other_command.explain as read-only and data-returning without inspecting the
// inner statement; we preserve that via a lexical EXPLAIN check.
func validateQuery(statement string) (bool, bool, error) {
	// Split into top-level statements with the omni splitter so EXPLAIN (which
	// omni cannot parse) can be classified per-segment before parsing.
	stmts, err := SplitSQL(statement)
	if err != nil {
		return false, false, err
	}

	returnsData := true
	for _, stmt := range stmts {
		if stmt.Empty || strings.TrimSpace(stmt.Text) == "" {
			continue
		}

		file, perr := parser.Parse(stmt.Text)
		if perr != nil {
			return false, false, perr
		}
		if file == nil || len(file.Stmts) == 0 {
			// A segment that parses to no statement (e.g. comment-only) is neither
			// read-only nor a write; treat it as a no-op like the legacy walker did.
			continue
		}

		readOnly, isSet := classifyForEditor(getQueryType(file.Stmts[0]), file.Stmts[0])
		if !readOnly {
			return false, false, nil
		}
		if isSet {
			returnsData = false
		}
	}

	return true, returnsData, nil
}

// classifyForEditor maps a statement's base.QueryType (plus its node, to single
// out SET) onto the SQL-editor read-only decision, mirroring the legacy
// queryValidateListener.EnterSql_command.
//
// Returns (readOnly, isSet):
//   - readOnly: the statement may run in the read-only SQL editor;
//   - isSet:    the statement is a SET (read-only but not data-returning).
//
// The read-only set is SELECT (base.Select for SELECT/set-op/USE/SET) and
// SHOW/DESCRIBE (base.SelectInfoSchema). However getQueryType maps USE to
// base.Select while the legacy editor listener REJECTED USE (use_command was
// not in its accepted dml/other/describe/show set), so USE is excluded here.
func classifyForEditor(qt base.QueryType, node ast.Node) (bool, bool) {
	if sh, ok := node.(*ast.ShowStmt); ok && sh.Pipe != nil {
		// SHOW ... ->> <stmt>: SHOW itself is read-only metadata; the verdict is
		// the piped statement's (a non-read-only pipe must not classify as a query).
		return classifyForEditor(getQueryType(sh.Pipe), sh.Pipe)
	}
	if rs, ok := node.(*ast.ResultScanStmt); ok {
		// stmt ->> query is read-only only if BOTH the source statement and the
		// trailing query are; it returns the trailing query's data.
		srcOK, _ := classifyForEditor(getQueryType(rs.Source), rs.Source)
		queryOK, querySet := classifyForEditor(getQueryType(rs.Query), rs.Query)
		return srcOK && queryOK, querySet
	}
	if _, ok := node.(*ast.UseStmt); ok {
		// Legacy validateQuery rejected USE even though getQueryType classifies it
		// read-only; preserve the stricter editor behavior.
		return false, false
	}
	if _, ok := node.(*ast.SetStmt); ok {
		// SET is read-only but does not return data.
		return true, true
	}
	switch qt {
	case base.Select, base.SelectInfoSchema, base.Explain:
		return true, false
	default:
		return false, false
	}
}
