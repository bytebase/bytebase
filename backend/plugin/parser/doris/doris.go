package doris

import (
	"github.com/bytebase/omni/doris/ast"
	"github.com/bytebase/omni/doris/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_DORIS, parseDorisStatements)
}

type astNode struct {
	node     ast.Node
	text     string
	startPos *storepb.Position
}

func (a *astNode) ASTStartPosition() *storepb.Position { return a.startPos }

func (a *astNode) Node() ast.Node { return a.node }

func parseDorisStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseSegmentedStatements(statement, SplitSQL, parseSegment)
}

func parseDorisSQL(statement string) ([]*astNode, error) {
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
	return &astNode{node: node, text: stmt.Text, startPos: stmt.Start}, nil
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
