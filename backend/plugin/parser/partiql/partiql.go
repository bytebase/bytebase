package partiql

import (
	"strings"

	"github.com/bytebase/omni/partiql/ast"
	"github.com/bytebase/omni/partiql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_DYNAMODB, parsePartiQLStatements)
}

// omniAST wraps an omni AST node to implement the base.AST interface.
type omniAST struct {
	node     ast.Node
	startPos *storepb.Position
}

func (a *omniAST) ASTStartPosition() *storepb.Position {
	return a.startPos
}

// parsePartiQLStatements is the ParseStatementsFunc for PartiQL (DynamoDB).
// Returns []ParsedStatement with both text and AST populated.
func parsePartiQLStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions.
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty {
			text := strings.TrimSpace(stmt.Text)
			if text == "" {
				ps.Empty = true
			} else {
				list, err := parser.Parse(text)
				if err != nil {
					return nil, err
				}
				var node ast.Node
				if len(list.Items) > 0 {
					node = list.Items[0]
				}
				ps.AST = &omniAST{
					node:     node,
					startPos: stmt.Start,
				}
			}
		}
		result = append(result, ps)
	}

	return result, nil
}
