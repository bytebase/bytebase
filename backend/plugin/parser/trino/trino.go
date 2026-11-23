// Package trino provides a SQL parser adapter for Trino, backed by the omni
// hand-written Trino parser (github.com/bytebase/omni/trino/*).
package trino

import (
	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_TRINO, parseTrinoStatements)
}

type astNode struct {
	node     ast.Node
	startPos *storepb.Position
}

func (a *astNode) ASTStartPosition() *storepb.Position { return a.startPos }

func (a *astNode) Node() ast.Node { return a.node }

func parseTrinoStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseSegmentedStatements(statement, SplitSQL, parseSegment)
}

func parseTrinoSQL(statement string) ([]*astNode, error) {
	return base.ParseNonEmptySegments(statement, SplitSQL, parseSegment)
}

func parseSegment(stmt base.Statement) (*astNode, error) {
	// Parse each segment on its own so ParseError byte offsets align with stmt.Text
	// before convertParseError shifts them into script coordinates.
	file, err := parser.Parse(stmt.Text)
	if err != nil {
		return nil, convertParseError(stmt.Text, err, stmt.Start)
	}
	var node ast.Node
	if file != nil && len(file.Stmts) > 0 {
		node = file.Stmts[0]
	}
	return &astNode{node: node, startPos: stmt.Start}, nil
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
