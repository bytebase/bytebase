package mongodb

import (
	"github.com/bytebase/omni/mongo"
	"github.com/bytebase/omni/mongo/ast"
	omniparser "github.com/bytebase/omni/mongo/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MONGODB, ParseMongoShell)
}

// AST wraps an omni MongoDB AST node and implements the base.AST interface.
type AST struct {
	Node          ast.Node
	StartPosition *storepb.Position
}

// ASTStartPosition implements base.AST.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// GetNode extracts the omni AST node from a base.AST interface.
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

// ParseMongoShell parses a MongoDB shell script and returns parsed statements
// with their ASTs. Conforms to the standard ParseStatementsFunc interface.
//
// Parsing is strict: a parse error in any statement fails the whole input so
// callers never silently lose statements (BYT-9950). Use
// ParseMongoShellBestEffort for partial results.
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
	positionMapper := base.NewByteOffsetPositionMapper(input)
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		baseStmt := base.NewExactStatement(input, positionMapper, stmt.ByteStart, stmt.ByteEnd, false)
		result = append(result, base.ParsedStatement{
			Statement: baseStmt,
			AST: &AST{
				Node:          stmt.AST,
				StartPosition: baseStmt.Start,
			},
		})
	}
	return result
}
