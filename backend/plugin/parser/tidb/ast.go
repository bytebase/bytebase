package tidb

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/bytebase/omni/tidb/ast"
	omniparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// parseTiDBStatements is the registered ParseStatementsFunc for TiDB (see
// tidb.go init). omni is the sole TiDB parser: the input is split and every
// statement is parsed with omni. A statement omni rejects surfaces as a
// base.SyntaxError with coordinates relative to the original input; there is
// no pingcap fallback.
//
// Accepted limitation: a little TiDB grammar newer than the pinned TiDB
// v8.5.5 that omni does not yet accept (notably TRAFFIC CAPTURE and COLUMNAR
// INDEX, which pingcap-master still parses) now surfaces as a parse error
// rather than being parsed by pingcap. Closing those omni gaps is follow-up.
func parseTiDBStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_TIDB, statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			result = append(result, base.ParsedStatement{Statement: stmt})
			continue
		}

		list, err := ParseTiDB(stmt.Text)
		if err != nil {
			return nil, convertParseError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			// Omni succeeded but produced no items (e.g. a comment-only
			// statement that survived the splitter). Preserve the statement
			// position with a nil AST.
			result = append(result, base.ParsedStatement{Statement: stmt})
			continue
		}
		for _, node := range list.Items {
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &AST{
					Node:          node,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}

// convertParseError takes an omni parser error and produces a
// base.SyntaxError with absolute line/column coordinates (so the customer
// sees a position relative to the original multi-statement input, not the
// per-statement excerpt). Returns the original error unchanged if it is not
// an *omniparser.ParseError.
func convertParseError(err error, stmt base.Statement) error {
	var parseErr *omniparser.ParseError
	if !errors.As(err, &parseErr) {
		return err
	}

	pos := ByteOffsetToRunePosition(stmt.Text, parseErr.Position)
	if stmt.Start != nil {
		pos.Line += stmt.Start.Line - 1
	}

	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, parseErr.Message)
	if parseErr.RelatedText != "" {
		msg += "\nrelated text: " + parseErr.RelatedText
	}

	return &base.SyntaxError{
		Position:   pos,
		Message:    msg,
		RawMessage: parseErr.Message,
	}
}

// AST wraps an omni/tidb AST node and implements the base.AST interface.
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

// ParseTiDB parses SQL using omni/tidb's parser and returns an ast.List.
// This is the recommended entry point for new code that needs omni/tidb AST
// nodes. Callers that need a pre-split []base.ParsedStatement should use the
// registered base.ParseStatements with storepb.Engine_TIDB instead.
func ParseTiDB(sql string) (*ast.List, error) {
	return omniparser.Parse(sql)
}

// GetNode extracts the omni AST node from a base.AST interface.
// Returns the node and true if it is an *AST, nil and false otherwise.
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

// ByteOffsetToRunePosition converts a byte offset in sql to a 1-based
// line:column position, where column is measured in Unicode code points
// (runes), matching storepb.Position semantics.
//
// byteOffset values past len(sql) are clamped to len(sql). Offsets that land
// inside a multi-byte UTF-8 sequence are treated as if they pointed at the
// start of the enclosing rune (they advance 0 runes past the last complete
// rune). Offsets before 0 are treated as 0.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}

	line := int32(1)
	runeCol := int32(0) // 0-based rune count on current line
	i := 0
	for i < byteOffset {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if i+size > byteOffset {
			// Offset lands inside a multi-byte rune. Treat it as pointing at
			// the start of the enclosing rune — don't count it.
			break
		}
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
		Column: runeCol + 1, // convert to 1-based
	}
}
