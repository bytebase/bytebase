package dynamodb

import (
	"strings"

	"github.com/bytebase/omni/dynamodb/analysis"
	"github.com/bytebase/omni/dynamodb/ast"
	"github.com/bytebase/omni/dynamodb/completion"
	"github.com/bytebase/omni/dynamodb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, completion.Engine)
	engine.Parse = parsePartiQLStatements
	base.Register(engine, storepb.Engine_DYNAMODB)
}

// astNode wraps an omni AST node to implement the base.AST interface.
type astNode = base.NodeAST[ast.Node]

// parsePartiQLStatements is the ParseStatementsFunc for PartiQL (DynamoDB).
// Returns []ParsedStatement with both text and AST populated.
func parsePartiQLStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_DYNAMODB, statement)
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
					return nil, convertParseError(stmt.Text, err, stmt.Start)
				}
				var node ast.Node
				if len(list.Items) > 0 {
					node = list.Items[0]
				}
				ps.AST = &astNode{
					Node:          node,
					StartPosition: stmt.Start,
				}
			}
		}
		result = append(result, ps)
	}

	return result, nil
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	return base.ConvertParseError(statement, err, basePos, func(pe *parser.ParseError) (int, string, string) {
		return pe.Position, pe.Error(), pe.Message
	})
}
