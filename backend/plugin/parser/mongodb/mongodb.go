package mongodb

import (
	"unicode/utf8"

	"github.com/bytebase/omni/mongo"
	"github.com/bytebase/omni/mongo/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MONGODB, ParseMongoShell)
}

// OmniAST wraps an omni MongoDB AST node and implements the base.AST interface.
type OmniAST struct {
	Node          ast.Node
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
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

// ParseMongoShell parses a MongoDB shell script and returns parsed statements
// with their ASTs. Conforms to the standard ParseStatementsFunc interface.
//
// TODO(bytebase/omni): The omni parser does not support error recovery — it stops
// at the first syntax error and returns zero statements. This means LSP-facing
// callers (statement ranges, diagnostics) lose all results when any single
// statement has a syntax error (e.g. user is mid-typing). Error recovery should
// be added to omni's mongo.Parse() so partial results are returned alongside errors,
// similar to how ANTLR's error recovery worked.
func ParseMongoShell(statement string) ([]base.ParsedStatement, error) {
	stmts, err := mongo.Parse(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}

		startPos := byteOffsetToPosition(statement, stmt.ByteStart)
		endPos := byteOffsetToPosition(statement, stmt.ByteEnd)

		ps := base.ParsedStatement{
			Statement: base.Statement{
				Text:  stmt.Text,
				Start: startPos,
				End:   endPos,
				Range: &storepb.Range{
					Start: int32(stmt.ByteStart),
					End:   int32(stmt.ByteEnd),
				},
			},
			AST: &OmniAST{
				Node:          stmt.AST,
				StartPosition: startPos,
			},
		}
		result = append(result, ps)
	}

	return result, nil
}

// byteOffsetToPosition converts a byte offset to a 1-based line:column position
// where column is measured in Unicode code points.
func byteOffsetToPosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}

	line := int32(1)
	runeCol := int32(0)
	i := 0
	for i < byteOffset {
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
