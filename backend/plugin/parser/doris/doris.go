package doris

import (
	"strings"
	"unicode/utf8"

	"github.com/bytebase/omni/doris/ast"
	"github.com/bytebase/omni/doris/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_DORIS, parseDorisStatements)
}

// omniAST wraps an omni AST node to implement the base.AST interface.
type omniAST struct {
	node     ast.Node
	startPos *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *omniAST) ASTStartPosition() *storepb.Position {
	return a.startPos
}

// Node returns the underlying omni AST node. May be nil if the segment
// failed to parse cleanly but errors were tolerated by the caller.
func (a *omniAST) Node() ast.Node {
	return a.node
}

// parseDorisStatements is the ParseStatementsFunc for Doris.
// Returns []ParsedStatement with both text and AST populated.
//
// For non-empty segments, the AST is an *omniAST wrapping the omni AST node.
// The first parse error encountered (per-segment) is returned to the caller,
// matching the prior behaviour of bailing out on the first syntax error.
func parseDorisStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	result := make([]base.ParsedStatement, 0, len(stmts))
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if stmt.Empty || strings.TrimSpace(stmt.Text) == "" {
			result = append(result, ps)
			continue
		}
		// Parse the segment text on its own so byte offsets in any ParseError
		// align with stmt.Text; convertParseError then shifts them into the
		// coordinates of the original multi-statement script.
		file, errs := parser.Parse(stmt.Text)
		if len(errs) > 0 {
			pe := errs[0]
			return nil, convertParseError(stmt.Text, &pe, stmt.Start)
		}
		var node ast.Node
		if file != nil && len(file.Stmts) > 0 {
			node = file.Stmts[0]
		}
		ps.AST = &omniAST{
			node:     node,
			startPos: stmt.Start,
		}
		result = append(result, ps)
	}

	return result, nil
}

// parseDorisSQL parses the given SQL statement using the omni Doris parser.
// Returns one *omniAST per non-empty segment (empty / comment-only segments
// are skipped).
//
// This retains the historical signature shape used by other doris package
// files; on the first parse error it returns a *base.SyntaxError with the
// position translated into the coordinates of the original input.
func parseDorisSQL(statement string) ([]*omniAST, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []*omniAST
	for _, stmt := range stmts {
		if stmt.Empty || strings.TrimSpace(stmt.Text) == "" {
			continue
		}
		file, errs := parser.Parse(stmt.Text)
		if len(errs) > 0 {
			pe := errs[0]
			return nil, convertParseError(stmt.Text, &pe, stmt.Start)
		}
		var node ast.Node
		if file != nil && len(file.Stmts) > 0 {
			node = file.Stmts[0]
		}
		result = append(result, &omniAST{
			node:     node,
			startPos: stmt.Start,
		})
	}
	return result, nil
}

// convertParseError converts an omni *parser.ParseError to a
// *base.SyntaxError that sql_service.go recognises for structured
// error diagnostics. It converts the byte offset in ParseError.Loc
// to 1-based line and 1-based column (rune-based) matching the
// storepb.Position convention used across other omni parser adapters.
//
// If basePos is non-nil, the computed position is offset by it so that
// errors from parsing an isolated statement segment are reported in the
// coordinates of the original multi-statement script.
func convertParseError(statement string, pe *parser.ParseError, basePos *storepb.Position) *base.SyntaxError {
	line, col := byteOffsetToPosition(statement, pe.Loc.Start)
	if basePos != nil {
		// The first line of the segment shares a line with basePos, so
		// column offsets only apply when the error is on that line.
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

// byteOffsetToPosition converts a 0-based byte offset to a 1-based line and
// 1-based column (in runes).
func byteOffsetToPosition(s string, offset int) (line, col int) {
	line = 1
	col = 1
	for i := 0; i < offset && i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		i += size
	}
	return line, col
}
