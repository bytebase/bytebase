package redshift

import (
	"fmt"
	"log/slog"
	"strings"

	redshiftast "github.com/bytebase/omni/redshift/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_REDSHIFT, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Redshift, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	stmt, err := statementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		// To handle cases where there are comments in the query.
		// eg. select * from t1 -- this is comment;
		// Add two new line symbols here.
		return fmt.Sprintf("WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;", base.TrimStatement(statement), limit)
	}
	return stmt
}

func statementWithResultLimitInline(statement string, limitCount int) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", errors.New("empty statement")
	}

	stmts, err := ParseRedshift(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}
	if len(stmts) != 1 {
		return "", errors.Errorf("expected exactly one statement, got %d", len(stmts))
	}

	sel, ok := stmts[0].AST.(*redshiftast.SelectStmt)
	if !ok {
		// Non-SELECT statement, return as-is.
		return statement, nil
	}

	return rewriteSelectLimit(statement, sel, limitCount)
}

// rewriteSelectLimit adds or adjusts the LIMIT clause of a SELECT statement
// using byte-offset positions from the omni AST to surgically edit the original SQL text.
func rewriteSelectLimit(sql string, sel *redshiftast.SelectStmt, limitCount int) (string, error) {
	if sel.LimitCount != nil {
		// Already has LIMIT — replace the value if ours is lower. extractIntFromNode
		// returns 0 both for a genuine LIMIT 0 and for an expression it cannot read
		// (e.g. LIMIT (1+2)), so a zero is trusted as "already stricter" only when
		// loc also proves the node is a literal A_Const — the same one the
		// replacement below can locate. A positive value never had this ambiguity.
		existingLimit := extractIntFromNode(sel.LimitCount)
		loc := nodeLocOf(sel.LimitCount)
		isLiteral := loc.Start >= 0
		if (existingLimit > 0 || (existingLimit == 0 && isLiteral)) && existingLimit <= limitCount {
			return sql, nil // existing limit is already lower or equal, keep it
		}
		if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(sql) {
			return sql[:loc.Start] + fmt.Sprintf("%d", limitCount) + sql[loc.End:], nil
		}
		// LimitCount is a non-constant expression (e.g. LIMIT $1, LIMIT (1+2)).
		// Cannot safely rewrite in-place; let the caller fall back to CTE wrapper.
		return "", errors.Errorf("cannot rewrite non-constant LIMIT expression")
	}

	// No LIMIT clause — find the right insertion point.
	// Redshift's PostgreSQL-derived grammar order: ... ORDER BY ... LIMIT ... FOR UPDATE ...
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
func findLimitInsertPosition(sel *redshiftast.SelectStmt) (int, bool) {
	// LIMIT must appear before FOR UPDATE/SHARE.
	if sel.LockingClause != nil {
		span := redshiftast.ListSpan(sel.LockingClause)
		if span.Start > 0 {
			return span.Start, true
		}
	}

	// Otherwise insert at SelectStmt.Loc.End (after everything, including outer parens).
	// For CTE queries, omni may report SelectStmt.Loc.End before the outer ORDER BY,
	// so account for the sort clause explicitly.
	end := sel.Loc.End
	if span := redshiftast.ListSpan(sel.SortClause); span.End > end {
		end = span.End
	}
	if end <= 0 {
		return 0, false
	}
	return end, false
}

// extractIntFromNode extracts an integer value from a LIMIT/OFFSET node.
func extractIntFromNode(node redshiftast.Node) int {
	switch n := node.(type) {
	case *redshiftast.Integer:
		return int(n.Ival)
	case *redshiftast.A_Const:
		if iv, ok := n.Val.(*redshiftast.Integer); ok {
			return int(iv.Ival)
		}
		return 0
	default:
		return 0
	}
}

// nodeLocOf returns the Loc of a node, handling common wrapper types.
func nodeLocOf(node redshiftast.Node) redshiftast.Loc {
	switch n := node.(type) {
	case *redshiftast.A_Const:
		return n.Loc
	default:
		return redshiftast.Loc{Start: -1, End: -1}
	}
}
