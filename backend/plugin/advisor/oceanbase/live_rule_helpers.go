package oceanbase

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

// oceanbaseNodes returns the statements that parsed to an omni node, with
// their nodes, index for index.
func oceanbaseNodes(parsed []base.ParsedStatement) ([]ast.Node, []base.ParsedStatement) {
	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}
	return nodes, stmts
}

// statementStartPosition is the position of the statement's first
// non-blank character.
func statementStartPosition(stmt base.ParsedStatement) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(oceanbaseparser.ByteOffsetToRunePosition(stmt.Text, contentStartIndex(stmt.Text)).Line))
}

// lineOfFinding is the line of the node a finding anchors, or of the
// statement's first line when it anchors none, as lineOf computes it.
func lineOfFinding(stmt base.ParsedStatement, f v1review.Finding) int {
	loc := ast.Loc{Start: f.Start, End: f.End}
	if f.Start == 0 && f.End == 0 {
		loc.Start = -1
	}
	return lineOf(stmt.BaseLine(), stmt.Text, loc)
}

// findingPosition is the position of the node a finding anchors, or of the
// statement's first line when it anchors none, as lineOf computes it.
func findingPosition(stmt base.ParsedStatement, f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(lineOfFinding(stmt, f))
}

// dryRunText is the statement as the dry run EXPLAINs and prints it: trimmed,
// with exactly one trailing semicolon.
func dryRunText(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ";") + ";"
}

// nodePosition is the position of an INSERT, UPDATE or DELETE node, as
// lineOf computes it.
func nodePosition(stmt base.ParsedStatement, node ast.Node) *storepb.Position {
	loc := ast.Loc{Start: -1}
	switch n := node.(type) {
	case *ast.InsertStmt:
		loc = n.Loc
	case *ast.UpdateStmt:
		loc = n.Loc
	case *ast.DeleteStmt:
		loc = n.Loc
	default:
	}
	return base.ConvertANTLRLineToPosition(lineOf(stmt.BaseLine(), stmt.Text, loc))
}

// explainRows EXPLAINs the statement text as the row-limit rules do and reads
// the estimated rows from the plan.
func explainRows(ctx context.Context, driver *sql.DB, text string) v1review.Probe {
	res, err := advisor.Query(ctx, advisor.QueryContext{}, driver, storepb.Engine_OCEANBASE, fmt.Sprintf("EXPLAIN format=json %s", text))
	if err != nil {
		return v1review.Probe{Error: err.Error()}
	}
	rows, err := getEstimatedRowsFromJSON(res)
	if err != nil {
		return v1review.Probe{CountError: err.Error()}
	}
	return v1review.Probe{Rows: rows}
}
