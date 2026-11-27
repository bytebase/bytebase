package cockroachdb

import (
	"github.com/bytebase/omni/cockroachdb/analysis"
	crdbcompletion "github.com/bytebase/omni/cockroachdb/completion"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, crdbcompletion.Engine)
	engine.Parse = parseCockroachDBStatements
	engine.StatementTypes = GetStatementTypes
	base.Register(engine, storepb.Engine_COCKROACHDB)
}

// AST is the AST implementation for CockroachDB parser.
// It implements the base.AST interface.
type AST struct {
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
	Stmt          analysis.ParsedStatement
}

// ASTStartPosition implements base.AST interface.
func (a *AST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ASTNode returns the parsed statement, the node omni's CockroachDB engine reads.
func (a *AST) ASTNode() any {
	return a.Stmt
}

// parseCockroachDBStatements is the ParseStatementsFunc for CockroachDB.
// Returns []ParsedStatement with both text and AST populated.
func parseCockroachDBStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions (uses PostgreSQL splitter).
	stmts, err := base.SplitMultiSQL(storepb.Engine_COCKROACHDB, statement)
	if err != nil {
		return nil, err
	}

	parsedStmts, err := analysis.ParseScript(statement)
	if err != nil {
		return nil, err
	}

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

// GetStatementTypes returns the types of the statements that have one, in order, each followed by
// the types of the mutations in its CTEs and statement sources, each type once per statement.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	stmts, err := statementsFromASTs(asts)
	if err != nil {
		return nil, err
	}
	return base.OmniStatementTypes(analysis.StatementTypesFromParsed(stmts)), nil
}

func statementsFromASTs(asts []base.AST) ([]analysis.ParsedStatement, error) {
	stmts := make([]analysis.ParsedStatement, 0, len(asts))
	for _, ast := range asts {
		crdbAST, ok := ast.(*AST)
		if !ok {
			return nil, errors.New("expected CockroachDB AST")
		}
		stmts = append(stmts, crdbAST.Stmt)
	}
	return stmts, nil
}
