package pg

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_POSTGRES, statementWithResultLimit)
	// CockroachDB parses through the PostgreSQL grammar here as it does for the
	// read-only gate and EXPLAIN (see query.go, explain.go).
	base.RegisterResultLimitFunc(storepb.Engine_COCKROACHDB, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for PostgreSQL and
// CockroachDB, which share this grammar. Neither takes an engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	stmt, err := statementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		// Fallback to CTE approach for problematic queries
		return fmt.Sprintf("WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;", base.TrimStatement(statement), limit)
	}
	return stmt
}

func statementWithResultLimitInline(statement string, limitCount int) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", errors.New("empty statement")
	}

	stmts, err := ParsePg(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	if len(stmts) != 1 {
		return "", errors.Errorf("expected exactly one statement, got %d", len(stmts))
	}

	sel, ok := stmts[0].AST.(*ast.SelectStmt)
	if !ok {
		// Non-SELECT statement, return as-is.
		return statement, nil
	}

	return rewriteSelectLimit(statement, sel, limitCount)
}

// rewriteSelectLimit adds or adjusts the LIMIT clause of a SELECT statement
// using byte-offset positions from the omni AST to surgically edit the original SQL text.
func rewriteSelectLimit(sql string, sel *ast.SelectStmt, limitCount int) (string, error) {
	if sel.LimitCount != nil {
		// Already has LIMIT — replace the value if ours is lower.
		existingLimit := extractIntFromNode(sel.LimitCount)
		if existingLimit > 0 && existingLimit <= limitCount {
			return sql, nil // existing limit is already lower or equal, keep it
		}
		loc := nodeLocOf(sel.LimitCount)
		if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(sql) {
			return sql[:loc.Start] + fmt.Sprintf("%d", limitCount) + sql[loc.End:], nil
		}
		// LimitCount is a non-constant expression (e.g. LIMIT $1, LIMIT (1+2)).
		// Cannot safely rewrite in-place; let the caller fall back to CTE wrapper.
		return "", errors.Errorf("cannot rewrite non-constant LIMIT expression")
	}

	// No LIMIT clause — find the right insertion point.
	// PostgreSQL grammar order: ... ORDER BY ... LIMIT ... FOR UPDATE ...
	// LIMIT goes BEFORE FOR UPDATE but AFTER everything else.
	insertPos, beforeLocking := findLimitInsertPosition(sel)
	if beforeLocking {
		// Inserting at the start of FOR UPDATE/SHARE. The original whitespace
		// before FOR becomes the separator before LIMIT; we add a trailing
		// space to separate the limit value from FOR.
		return sql[:insertPos] + fmt.Sprintf("LIMIT %d ", limitCount) + sql[insertPos:], nil
	}
	return sql[:insertPos] + fmt.Sprintf(" LIMIT %d", limitCount) + sql[insertPos:], nil
}

// findLimitInsertPosition returns the byte offset where " LIMIT N" should be inserted,
// and whether the insertion is before a locking clause (FOR UPDATE/SHARE).
func findLimitInsertPosition(sel *ast.SelectStmt) (int, bool) {
	// LIMIT must appear before FOR UPDATE/SHARE.
	if sel.LockingClause != nil {
		span := ast.ListSpan(sel.LockingClause)
		if span.Start > 0 {
			return span.Start, true
		}
	}

	// Otherwise insert at SelectStmt.Loc.End (after everything, including outer parens).
	// For CTE queries, omni may report SelectStmt.Loc.End before the outer ORDER BY,
	// so account for the sort clause explicitly.
	end := sel.Loc.End
	if span := ast.ListSpan(sel.SortClause); span.End > end {
		end = span.End
	}
	if end <= 0 {
		return 0, false
	}
	return end, false
}

// extractIntFromNode extracts an integer value from a LIMIT/OFFSET node.
func extractIntFromNode(node ast.Node) int {
	switch n := node.(type) {
	case *ast.Integer:
		return int(n.Ival)
	case *ast.A_Const:
		if iv, ok := n.Val.(*ast.Integer); ok {
			return int(iv.Ival)
		}
		return 0
	default:
		return 0
	}
}

// nodeLocOf returns the Loc of a node, handling common wrapper types.
func nodeLocOf(node ast.Node) ast.Loc {
	switch n := node.(type) {
	case *ast.A_Const:
		return n.Loc
	default:
		return ast.Loc{Start: -1, End: -1}
	}
}
