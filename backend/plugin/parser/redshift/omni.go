package redshift

import (
	"errors"
	"fmt"
	"unicode/utf8"

	omniredshift "github.com/bytebase/omni/redshift"
	"github.com/bytebase/omni/redshift/ast"
	omniredshiftparser "github.com/bytebase/omni/redshift/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni AST node and implements the base.AST interface.
type OmniAST struct {
	// Node is the omni AST node (e.g. *ast.SelectStmt, *ast.CreateStmt).
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseRedshiftOmni parses SQL using omni's parser.
func ParseRedshiftOmni(sql string) ([]omniredshift.Statement, error) {
	return omniredshift.Parse(sql)
}

// GetOmniNode extracts the omni AST node from a base.AST interface.
func GetOmniNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	omniAST, ok := a.(*OmniAST)
	if !ok {
		return nil, false
	}
	return omniAST.Node, true
}

func convertOmniError(err error, stmt base.Statement) error {
	var parseErr *omniredshiftparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	position := ByteOffsetToRunePosition(stmt.Text, parseErr.Position)
	if stmt.Start != nil {
		localLine := position.Line
		position.Line += stmt.Start.Line - 1
		if localLine == 1 {
			position.Column += stmt.Start.Column - 1
		}
	}
	return &base.SyntaxError{
		Position:   position,
		Message:    fmt.Sprintf("Syntax error at line %d: %s", position.Line, parseErr.Message),
		RawMessage: parseErr.Message,
	}
}

func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}

	line := int32(1)
	runeCol := int32(0)
	for i := 0; i < byteOffset; {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '\n' {
			line++
			runeCol = 0
		} else {
			runeCol++
		}
		i += size
	}

	return &storepb.Position{
		Line:   line,
		Column: runeCol + 1,
	}
}
