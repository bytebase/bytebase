package mariadb

import (
	"github.com/bytebase/omni/mariadb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps an omni MariaDB AST node and implements the base.AST interface.
type AST struct {
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTSource returns the statement's own text, which the schema walk-through reads
// when it needs the statement as written.
func (a *AST) ASTSource() string {
	return a.Text
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetNode extracts the omni AST node from a base.AST interface.
// Returns the node and true if it is an AST, nil and false otherwise.
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
