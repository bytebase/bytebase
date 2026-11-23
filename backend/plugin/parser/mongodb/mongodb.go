package mongodb

import (
	"github.com/bytebase/omni/mongo"
	"github.com/bytebase/omni/mongo/ast"
	omniparser "github.com/bytebase/omni/mongo/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MONGODB, parseMongoShell)
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

// parseMongoShell parses a MongoDB shell script strictly so a parse error in any
// statement fails the whole input instead of silently dropping statements (BYT-9950).
func parseMongoShell(statement string) ([]base.ParsedStatement, error) {
	stmts, err := mongo.Parse(statement)
	if err != nil {
		return nil, err
	}
	return convertStatements(statement, stmts), nil
}

// parseMongoShellBestEffort returns partial statements and parse errors for LSP callers.
func parseMongoShellBestEffort(statement string) ([]base.ParsedStatement, []*omniparser.ParseError) {
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
