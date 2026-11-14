// Package tidb implements the SQL advisor rules for TiDB.
package tidb

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	omniast "github.com/bytebase/omni/tidb/ast"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/store/model"
)

// Stmt bundles an omni/tidb AST node with the source text and base line of
// the statement it came from. The base line is needed to convert byte offsets
// inside Text into absolute line numbers in the original full SQL.
type Stmt struct {
	Node     omniast.Node
	Text     string
	BaseLine int // 0-based line index of the first line of Text in the original SQL
}

// AbsoluteLine converts a byte offset within s.Text into a 1-based line number
// in the original SQL.
func (s Stmt) AbsoluteLine(byteOffset int) int {
	pos := tidbparser.ByteOffsetToRunePosition(s.Text, byteOffset)
	return s.BaseLine + int(pos.Line)
}

// TrimmedText returns Text with surrounding whitespace removed. Suitable
// for embedding the statement text into advice content; raw Text may
// include leading/trailing newlines from the original multi-statement
// split.
func (s Stmt) TrimmedText() string {
	return strings.TrimSpace(s.Text)
}

// FirstTokenLine returns the 1-based absolute line of the first
// non-whitespace character in s.Text. Matches pingcap's
// OriginTextPosition: pingcap's lexer strips leading whitespace but
// keeps comments as part of the statement, so its reported line points
// at the first comment OR keyword. Used as the StartPosition for
// statement-level advices.
func (s Stmt) FirstTokenLine() int {
	for i, r := range s.Text {
		if !unicode.IsSpace(r) {
			return s.AbsoluteLine(i)
		}
	}
	return s.AbsoluteLine(0)
}

// isRandFuncCall checks whether the given expression is a
// function call to RAND(). Used by insert_disallow_order_by_rand.
// Pingcap-tidb compared via `FnName.L == ast.Rand` (lowercase
// canonicalization); omni keeps the user's original case in
// `FuncCallExpr.Name`, so we compare case-insensitively. RAND
// accepts an optional integer seed argument; we match regardless
// of arity (matching pingcap's predecessor — which type-asserted
// the function call but didn't inspect Args).
func isRandFuncCall(expr omniast.ExprNode) bool {
	if expr == nil {
		return false
	}
	fc, ok := expr.(*omniast.FuncCallExpr)
	if !ok {
		return false
	}
	return strings.EqualFold(fc.Name, "RAND")
}

// addColumnTargets returns the column definitions added by an ATAddColumn
// cmd. omni populates either cmd.Columns (multi-column ADD COLUMN (...))
// or cmd.Column (single ADD COLUMN); read both defensively.
func addColumnTargets(cmd *omniast.AlterTableCmd) []*omniast.ColumnDef {
	if cmd == nil {
		return nil
	}
	if len(cmd.Columns) > 0 {
		return cmd.Columns
	}
	if cmd.Column != nil {
		return []*omniast.ColumnDef{cmd.Column}
	}
	return nil
}

// stmtsCacheKey is the advisor.Context.Memo key for the per-review
// omni-parse result. All migrated tidb advisors share one parse pass through
// this cache.
const stmtsCacheKey = "tidb.stmts"

// getTiDBNodes returns omni-parsed statements for migrated advisors. The
// dispatcher has already parsed and populated stmt.AST; this helper unwraps the
// nodes and memoizes the result on checkCtx.Memo (one parse per review).
//
// Type switch over stmt.AST:
//   - *tidbparser.AST: omni accepted — collect the node.
//   - default: unknown AST type — warn (a future tidb AST type shouldn't
//     silently drop statements).
func getTiDBNodes(checkCtx advisor.Context) ([]Stmt, error) {
	if cached, ok := checkCtx.Memo(stmtsCacheKey); ok {
		if stmts, typeOK := cached.([]Stmt); typeOK {
			return stmts, nil
		}
	}

	if checkCtx.ParsedStatements == nil {
		return nil, errors.New("ParsedStatements is not provided in context")
	}

	var result []Stmt
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.Empty || stmt.AST == nil {
			continue
		}
		switch a := stmt.AST.(type) {
		case *tidbparser.AST:
			result = append(result, Stmt{
				Node:     a.Node,
				Text:     a.Text,
				BaseLine: stmt.BaseLine(),
			})
		default:
			slog.Warn("unexpected stmt.AST type for tidb dispatcher",
				slog.String("type", fmt.Sprintf("%T", a)))
		}
	}

	checkCtx.SetMemo(stmtsCacheKey, result)
	return result, nil
}

// stmtNodes returns the nodes of stmts, index for index, for a rule that
// judges the statements together.
func stmtNodes(stmts []Stmt) []omniast.Node {
	nodes := make([]omniast.Node, len(stmts))
	for i, stmt := range stmts {
		nodes[i] = stmt.Node
	}
	return nodes
}

// v1reviewSchema returns the snapshot md holds for an omni rule, an empty
// one when md is nil.
func v1reviewSchema(md *model.DatabaseMetadata) v1review.Schema {
	if md == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: md.GetProto(), ObjectCaseSensitive: md.GetIsObjectCaseSensitive()}
}
