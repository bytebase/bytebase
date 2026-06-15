package snowflake

import (
	"errors"
	"fmt"

	omniast "github.com/bytebase/omni/snowflake/ast"
	omniparser "github.com/bytebase/omni/snowflake/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var _ base.AST = (*OmniAST)(nil)

// OmniAST wraps an omni AST node and implements the base.AST interface.
// It mirrors redshift's OmniAST.
type OmniAST struct {
	// Node is the omni AST node for this statement (e.g. *ast.SelectStmt).
	// It is always non-nil for a successfully parsed non-empty statement.
	Node omniast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetOmniNode extracts the omni AST node from a base.AST interface.
// It returns false when a is not a snowflake OmniAST (or carries no node).
func GetOmniNode(a base.AST) (omniast.Node, bool) {
	if a == nil {
		return nil, false
	}
	omniAST, ok := a.(*OmniAST)
	if !ok || omniAST.Node == nil {
		return nil, false
	}
	return omniAST.Node, true
}

// parseOmniStatementNode parses ONE statement's text with the omni parser and
// returns its single top-level statement node. Unlike the dual-AST transition
// (where omni was best-effort behind the legacy gatekeeper), a parse failure
// is now a hard error: the omni parser is the only parser, so a statement it
// rejects must fail the batch the way a legacy ANTLR syntax error did.
func parseOmniStatementNode(text string) (omniast.Node, error) {
	file, err := parseSnowflakeAST(text)
	if err != nil {
		return nil, err
	}
	if file == nil || len(file.Stmts) == 0 {
		return nil, errors.New("statement yielded no parse tree")
	}
	return file.Stmts[0], nil
}

// convertOmniParseError converts an omni *parser.ParseError into the bare
// *base.SyntaxError shape the legacy ANTLR error listener produced (callers
// such as the sheet manager type-assert on *base.SyntaxError to surface
// syntax-error advices). The error position is computed inside the statement
// text (rune-based columns) and offset by the statement's start position in
// the original multi-statement input — mirroring redshift's convertOmniError.
func convertOmniParseError(err error, stmt base.Statement) error {
	var parseErr *omniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	mapper := base.NewByteOffsetPositionMapper(stmt.Text)
	position := mapper.Position(parseErr.Loc.Start)
	if stmt.Start != nil {
		localLine := position.Line
		position.Line += stmt.Start.Line - 1
		if localLine == 1 && stmt.Start.Column > 0 {
			position.Column += stmt.Start.Column - 1
		}
	}
	return &base.SyntaxError{
		Position:   position,
		Message:    fmt.Sprintf("Syntax error at line %d:%d \n%s", position.Line, position.Column, parseErr.Msg),
		RawMessage: parseErr.Msg,
	}
}
