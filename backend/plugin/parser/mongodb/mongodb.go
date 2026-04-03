package mongodb

import (
	"unicode/utf8"

	"github.com/bytebase/omni/mongo"
	"github.com/bytebase/omni/mongo/ast"
	omniparser "github.com/bytebase/omni/mongo/parser"

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
// The omni parser recovers from syntax errors by skipping to the next statement
// boundary. When at least one statement parses successfully, this function
// returns the partial results with a nil error. A non-nil error is returned
// only when no statements could be parsed at all.
func ParseMongoShell(statement string) ([]base.ParsedStatement, error) {
	stmts, err := mongo.Parse(statement)
	if err != nil {
		return nil, err
	}
	return convertStatements(statement, stmts), nil
}

// ParseMongoShellBestEffort parses a MongoDB shell script and returns as many
// successfully parsed statements as possible, along with any parse errors.
// Used by LSP-facing functions where partial results are better than nothing.
func ParseMongoShellBestEffort(statement string) ([]base.ParsedStatement, []*omniparser.ParseError) {
	result := mongo.ParseBestEffort(statement)
	return convertStatements(statement, result.Statements), result.Errors
}

func convertStatements(input string, stmts []mongo.Statement) []base.ParsedStatement {
	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		startPos := byteOffsetToPosition(input, stmt.ByteStart)
		endPos := byteOffsetToPosition(input, stmt.ByteEnd)
		result = append(result, base.ParsedStatement{
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
		})
	}
	return result
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
