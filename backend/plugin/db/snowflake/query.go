package snowflake

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	omniast "github.com/bytebase/omni/snowflake/ast"
	omniparser "github.com/bytebase/omni/snowflake/parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/utils"
)

const stmtErrFmt = "statement: %s"

func getStatementWithResultLimit(statement string, limit int) string {
	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", util.TrimStatement(statement), limit)
	}
	return stmt
}

// getStatementWithResultLimitInline rewrites the single statement so that its
// outermost query returns at most limitCount rows. The rewrite splices the
// replacement into the original statement text via omni AST byte offsets
// (ast.Loc), preserving the original formatting. The output shape mirrors the
// legacy ANTLR TokenStreamRewriter: the statement trimmed of trailing
// spaces/semicolons, plus a final "\n;".
//
// Mirroring the legacy rewriter, only the outermost query's right-most select
// is touched:
//   - LIMIT <n>            → n becomes min(n, limitCount) (n == 0 → limitCount)
//   - OFFSET/FETCH <n>     → the fetch count is replaced the same way
//   - TOP <n>              → replaced the same way
//   - no limiting clause   → " LIMIT <limitCount>" is appended at the end of
//     that select (inside any wrapping parentheses)
//
// Non-query statements are returned unchanged (the legacy listener never
// fired on them). Any error makes the caller fall back to wrapping the
// statement in SELECT * FROM (...) LIMIT n.
func getStatementWithResultLimitInline(singleStatement string, limitCount int) (string, error) {
	trimmed := strings.TrimRightFunc(singleStatement, utils.IsSpaceOrSemicolon)

	file, err := omniparser.Parse(trimmed)
	if err != nil {
		return "", errors.Wrapf(err, stmtErrFmt, singleStatement)
	}
	if len(file.Stmts) != 1 {
		return "", errors.Errorf("expected exactly 1 statement, got %d", len(file.Stmts))
	}

	sel := findRightMostSelect(file.Stmts[0])
	if sel == nil {
		// Not a query statement; return the normalized text unchanged.
		return trimmed + "\n;", nil
	}

	// LIMIT <n> [OFFSET <m>]: replace n.
	if sel.Limit != nil {
		spliced, err := spliceLimitNumber(trimmed, omniast.NodeLoc(sel.Limit), limitCount)
		if err != nil {
			return "", errors.Wrapf(err, stmtErrFmt, singleStatement)
		}
		return spliced + "\n;", nil
	}

	// [OFFSET <m>] FETCH [FIRST|NEXT] <n> ...: replace the fetch count n.
	if sel.Fetch != nil {
		spliced, err := spliceLimitNumber(trimmed, omniast.NodeLoc(sel.Fetch.Count), limitCount)
		if err != nil {
			return "", errors.Wrapf(err, stmtErrFmt, singleStatement)
		}
		return spliced + "\n;", nil
	}

	// A bare OFFSET without LIMIT/FETCH cannot take an appended LIMIT clause
	// (LIMIT must precede OFFSET); the legacy parser rejected this shape, so
	// keep routing it to the caller's fallback.
	if sel.Offset != nil {
		return "", errors.Errorf("statement has OFFSET without LIMIT/FETCH: %s", singleStatement)
	}

	// TOP <n>: replace n.
	if sel.Top != nil {
		spliced, err := spliceLimitNumber(trimmed, omniast.NodeLoc(sel.Top), limitCount)
		if err != nil {
			return "", errors.Wrapf(err, stmtErrFmt, singleStatement)
		}
		return spliced + "\n;", nil
	}

	// No limiting clause: append after the end of the select.
	end := sel.Loc.End
	if !sel.Loc.IsValid() || end > len(trimmed) {
		return "", errors.Errorf("invalid select location %v for statement: %s", sel.Loc, singleStatement)
	}
	return trimmed[:end] + fmt.Sprintf(" LIMIT %d", limitCount) + trimmed[end:] + "\n;", nil
}

// spliceLimitNumber replaces the number at loc within text by
// min(userLimit, limitCount), where a user limit of 0 means "no limit" and
// becomes limitCount — the exact arithmetic of the legacy rewriter. A
// non-integer at loc is an error (the legacy grammar only accepted integer
// tokens there; anything else fell back).
func spliceLimitNumber(text string, loc omniast.Loc, limitCount int) (string, error) {
	if !loc.IsValid() || loc.End > len(text) || loc.Start > loc.End {
		return "", errors.Errorf("invalid limit number location %v", loc)
	}
	userLimit, err := strconv.Atoi(text[loc.Start:loc.End])
	if err != nil {
		return "", errors.Wrapf(err, "non-integer limit count %q", text[loc.Start:loc.End])
	}
	limit := userLimit
	if limit == 0 || limitCount < limit {
		limit = limitCount
	}
	return text[:loc.Start] + strconv.Itoa(limit) + text[loc.End:], nil
}

// findRightMostSelect returns the right-most SELECT of the outermost query:
// for set operations the last operand's select (recursing through nested set
// operations and parenthesized selects, which omni flattens), for the
// result-pipe operator (->>) the consuming query's select. Returns nil for
// non-query statements.
func findRightMostSelect(node omniast.Node) *omniast.SelectStmt {
	switch n := node.(type) {
	case *omniast.SelectStmt:
		return n
	case *omniast.SetOperationStmt:
		return findRightMostSelect(n.Right)
	case *omniast.ResultScanStmt:
		return findRightMostSelect(n.Query)
	case *omniast.ShowStmt:
		// SHOW ... ->> <query>: omni stores the piped statement on ShowStmt.Pipe;
		// the row limit applies to that trailing query.
		if n.Pipe != nil {
			return findRightMostSelect(n.Pipe)
		}
		return nil
	default:
		return nil
	}
}
