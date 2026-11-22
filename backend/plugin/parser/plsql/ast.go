package plsql

import (
	"errors"
	"fmt"

	"github.com/bytebase/omni/oracle/analysis"
	"github.com/bytebase/omni/oracle/ast"
	oracleparser "github.com/bytebase/omni/oracle/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// AST wraps an omni AST node and implements the base.AST interface.
type AST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateTableStmt).
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

// ParsePLSQL parses SQL using omni's parser and returns an ast.List.
// This is the recommended entry point for new Oracle code that needs omni AST nodes.
func ParsePLSQL(sql string) (*ast.List, error) {
	return analysis.Parse(sql)
}

// GetNode extracts the omni AST node from a base.AST interface.
// Returns the node and true if it is an AST, nil and false otherwise.
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

// ByteOffsetToRunePosition converts a byte offset in sql to a 1-based line:column position.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	return base.ByteOffsetToRunePosition(sql, byteOffset)
}

func convertError(err error, stmt base.Statement) error {
	var parseErr *oracleparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}
	return convertParseError(parseErr, stmt)
}

func convertParseError(parseErr *oracleparser.ParseError, stmt base.Statement) *base.SyntaxError {
	pos := base.ShiftPosition(ByteOffsetToRunePosition(stmt.Text, parseErr.Position), stmt.Start)

	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, parseErr.Message)
	return &base.SyntaxError{
		Position:   pos,
		Message:    msg,
		RawMessage: parseErr.Message,
	}
}
