// Package trino provides a SQL parser adapter for Trino, backed by the omni
// hand-written Trino parser (github.com/bytebase/omni/trino/*).
package trino

import (
	"strings"

	"github.com/bytebase/omni/trino/ast"
	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_TRINO, parseTrinoStatements)
}

// omniAST wraps an omni Trino AST node so it satisfies the base.AST interface.
//
// The concrete statement node types (e.g. *parser.QueryStmt, *parser.InsertStmt)
// live in the omni trino/parser package and implement ast.Node; parser.Parse
// returns them inside *ast.File.Stmts.
type omniAST struct {
	node     ast.Node
	startPos *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *omniAST) ASTStartPosition() *storepb.Position {
	return a.startPos
}

// Node returns the underlying omni AST node. It may be nil if the segment
// produced no concrete statement (which should not happen for valid SQL).
func (a *omniAST) Node() ast.Node {
	return a.node
}

// parseTrinoStatements is the ParseStatementsFunc registered for Trino. It
// splits the input into top-level statements and parses each one, returning a
// []base.ParsedStatement carrying both the statement text/positions and the
// parsed AST.
//
// Each segment is parsed on its own so that any ParseError byte offsets align
// with the segment text; convertParseError then shifts them into the
// coordinates of the original multi-statement script. The first parse error
// encountered is returned, matching the prior behaviour of bailing out on the
// first syntax error.
func parseTrinoStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	result := make([]base.ParsedStatement, 0, len(stmts))
	for _, stmt := range stmts {
		parsed, err := parseSegment(stmt)
		if err != nil {
			return nil, err
		}
		ps := base.ParsedStatement{Statement: stmt}
		if parsed != nil {
			ps.AST = parsed
		}
		result = append(result, ps)
	}

	return result, nil
}

// parseTrinoSQL parses the given SQL using the omni Trino parser and returns one
// *omniAST per non-empty segment (empty / comment-only segments are skipped).
//
// On the first parse error it returns a *base.SyntaxError with the position
// translated into the coordinates of the original input. This mirrors the
// historical ParseTrino signature shape used by other files in this package.
func parseTrinoSQL(statement string) ([]*omniAST, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*omniAST
	for _, stmt := range stmts {
		parsed, err := parseSegment(stmt)
		if err != nil {
			return nil, err
		}
		if parsed != nil {
			result = append(result, parsed)
		}
	}
	return result, nil
}

// parseSegment parses a single already-split statement segment into an
// *omniAST. A blank or comment-only segment yields (nil, nil) so callers can
// decide whether to keep a placeholder (parseTrinoStatements) or skip it
// (parseTrinoSQL). The first omni parse error is translated into the
// coordinates of the original input via convertParseError.
func parseSegment(stmt base.Statement) (*omniAST, error) {
	if stmt.Empty || strings.TrimSpace(stmt.Text) == "" {
		return nil, nil
	}
	file, errs := parser.Parse(stmt.Text)
	if len(errs) > 0 {
		return nil, convertParseError(stmt.Text, &errs[0], stmt.Start)
	}
	var node ast.Node
	if file != nil && len(file.Stmts) > 0 {
		node = file.Stmts[0]
	}
	return &omniAST{node: node, startPos: stmt.Start}, nil
}

// convertParseError converts an omni *parser.ParseError to a *base.SyntaxError
// that sql_service.go recognises for structured error diagnostics. It converts
// the byte offset in ParseError.Loc to 1-based line and 1-based column
// (rune-based), matching the storepb.Position convention used across other omni
// parser adapters.
//
// If basePos is non-nil, the computed position is offset by it so that errors
// from parsing an isolated statement segment are reported in the coordinates of
// the original multi-statement script.
func convertParseError(statement string, pe *parser.ParseError, basePos *storepb.Position) *base.SyntaxError {
	pos := base.NewByteOffsetPositionMapper(statement).Position(pe.Loc.Start)
	line, col := int(pos.Line), int(pos.Column)
	if basePos != nil {
		if line == 1 {
			col = int(basePos.Column) + col - 1
		}
		line = int(basePos.Line) + line - 1
	}
	return &base.SyntaxError{
		Position: &storepb.Position{
			Line:   int32(line),
			Column: int32(col),
		},
		Message:    pe.Error(),
		RawMessage: pe.Msg,
	}
}

// NormalizeTrinoIdentifier normalizes a Trino identifier. Trino folds unquoted
// identifiers to lower case and preserves the case of double-quoted identifiers
// (stripping the surrounding quotes). This delegates to the omni normalizer so
// the bytebase adapter and the omni parser agree on identifier folding.
func NormalizeTrinoIdentifier(ident string) string {
	return parser.NormalizeTrinoIdentifier(ident)
}
