package tidb

import (
	"errors"
	"fmt"

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

// ByteOffsetToRunePosition converts a byte offset in sql to a 1-based line:column position.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	return base.ByteOffsetToRunePosition(sql, byteOffset)
}
