package partiql

import (
	"errors"
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
			if strings.TrimSpace(stmt.Text) == "" {
				ps.Empty = true
			} else {
				// Parse the segment as-is so that ParseError byte offsets
				// align with stmt.Text; convertParseError then offsets them
				// by stmt.Start to refer back to the original script.
				list, err := parser.Parse(stmt.Text)
				if err != nil {
					var parseErr *parser.ParseError
					if errors.As(err, &parseErr) {
						return nil, convertParseError(stmt.Text, parseErr, stmt.Start)
					}
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
