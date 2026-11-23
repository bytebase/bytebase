package tidb

import (
	"github.com/bytebase/omni/tidb/ast"
	omniparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func parseTiDBStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, func(s string) ([]base.Statement, error) {
		return base.SplitMultiSQL(storepb.Engine_TIDB, s)
	}, func(stmt base.Statement) ([]base.AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		list, err := ParseTiDB(stmt.Text)
		if err != nil {
			return nil, convertParseError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			return nil, nil
		}
		asts := make([]base.AST, len(list.Items))
		for i, node := range list.Items {
			asts[i] = &AST{
				Node:          node,
				Text:          stmt.Text,
				StartPosition: stmt.Start,
			}
		}
		return asts, nil
	})
}

func convertParseError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, false,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}

// AST wraps an omni/tidb AST node and implements the base.AST interface.
type AST struct {
	Node          ast.Node
	Text          string
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseTiDB parses SQL using omni/tidb's parser and returns an ast.List.
func ParseTiDB(sql string) (*ast.List, error) {
	return omniparser.Parse(sql)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	wrapped, ok := a.(*AST)
	if !ok || wrapped == nil {
		return nil, false
	}
	return wrapped.Node, true
}

// ByteOffsetToRunePosition converts a byte offset in sql to a 1-based line:column position.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	return base.ByteOffsetToRunePosition(sql, byteOffset)
}
