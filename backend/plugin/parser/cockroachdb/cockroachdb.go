package cockroachdb

import (
	"github.com/bytebase/omni/cockroachdb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_COCKROACHDB, parseCockroachDBStatements)
}

// parseCockroachDBStatements is the ParseStatementsFunc for CockroachDB.
// Returns []ParsedStatement with both text and AST populated.
func parseCockroachDBStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions (uses PostgreSQL splitter)
	stmts, err := base.SplitMultiSQL(storepb.Engine_COCKROACHDB, statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	parsedStmts, err := analysis.ParseScript(statement)
	if err != nil {
		return nil, err
	}

	// Combine: Statement provides text/positions, AST provides parsed tree
	var result []base.ParsedStatement
	astIndex := 0
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty && astIndex < len(parsedStmts) {
			ps.AST = &AST{
				StartPosition: &storepb.Position{Line: 1},
				Stmt:          parsedStmts[astIndex],
			}
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}
