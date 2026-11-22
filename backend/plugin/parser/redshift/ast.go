package redshift

import (
	"errors"
	"fmt"

	omniredshift "github.com/bytebase/omni/redshift"
	"github.com/bytebase/omni/redshift/ast"
	omniredshiftparser "github.com/bytebase/omni/redshift/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps an omni AST node and implements the base.AST interface.
type AST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseRedshift parses SQL using omni's parser.
func ParseRedshift(sql string) ([]omniredshift.Statement, error) {
	return omniredshift.Parse(sql)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	wrapped, ok := a.(*AST)
	if !ok {
		return nil, false
	}
	return wrapped.Node, true
}

func convertError(err error, stmt base.Statement) error {
	var parseErr *omniredshiftparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	position := base.ShiftPosition(ByteOffsetToRunePosition(stmt.Text, parseErr.Position), stmt.Start)
	return &base.SyntaxError{
		Position:   position,
		Message:    fmt.Sprintf("Syntax error at line %d: %s", position.Line, parseErr.Message),
		RawMessage: parseErr.Message,
	}
}

func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	return base.ByteOffsetToRunePosition(sql, byteOffset)
}
