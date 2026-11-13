package pg

import (
	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// parsedNodes returns the statements that carry an omni node and their nodes,
// index for index.
func parsedNodes(parsed []base.ParsedStatement) ([]base.ParsedStatement, []ast.Node) {
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
}

// findingLine returns the absolute line a V1 advisor reported a finding of
// stmt at: the line of the first occurrence of the finding's Search text in
// the statement, or the line the statement's content starts on.
func findingLine(stmt base.ParsedStatement, f v1review.Finding) int32 {
	rule := &BaseRule{StmtText: stmt.Text}
	line := rule.ContentStartLine()
	if f.Search != "" {
		line = rule.FindLineByName(f.Search)
	}
	return line + int32(stmt.BaseLine())
}
