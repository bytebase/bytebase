package cassandra

import (
	omnicassandra "github.com/bytebase/omni/cassandra"
	"github.com/bytebase/omni/cassandra/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps an omni AST node and implements the base.AST interface.
type AST struct {
	Node          ast.Node
	Text          string
	StartPosition *storepb.Position
}

func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	node, ok := a.(*AST)
	if !ok {
		return nil, false
	}
	return node.Node, true
}

// ParseCQL parses CQL using omni's parser and returns omni Statement objects.
func ParseCQL(sql string) ([]omnicassandra.Statement, error) {
	return omnicassandra.Parse(sql)
}

func byteOffsetToPosition(sql string, byteOffset int) *storepb.Position {
	return base.NewByteOffsetPositionMapper(sql).Position(byteOffset)
}
