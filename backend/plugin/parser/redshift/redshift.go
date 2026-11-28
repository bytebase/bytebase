package redshift

import (
	omniredshift "github.com/bytebase/omni/redshift"
	"github.com/bytebase/omni/redshift/analysis"
	"github.com/bytebase/omni/redshift/ast"
	redshiftcompletion "github.com/bytebase/omni/redshift/completion"
	omniredshiftparser "github.com/bytebase/omni/redshift/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, redshiftcompletion.Engine)
	engine.Parse = parseRedshiftStatements
	engine.StatementTypes = GetStatementTypes
	base.Register(engine, storepb.Engine_REDSHIFT)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

func parseRedshiftStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, splitSQL, func(stmt base.Statement) ([]base.AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		parsed, err := omniredshift.Parse(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, 0, len(parsed))
		for _, parsedStmt := range parsed {
			if parsedStmt.Empty() {
				continue
			}
			asts = append(asts, &AST{
				Node:          parsedStmt.AST,
				Text:          stmt.Text,
				StartPosition: stmt.Start,
			})
		}
		return asts, nil
	})
}

func getStatementTypesWithPosition(asts []base.AST) ([]base.StatementTypeWithPosition, error) {
	return base.CollectSingleStatementTypesWithPosition(asts, "expected Redshift omni AST", false, func(a *AST) (storepb.StatementType, string, bool) {
		if a == nil || a.Node == nil {
			return 0, "", false
		}
		return base.OmniStatementType(analysis.StatementTypeOf(a.Node)), a.Text, true
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := getStatementTypesWithPosition(asts)
	if err != nil {
		return nil, err
	}
	return base.ExtractTypesFromPositions(results), nil
}

func splitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_REDSHIFT, statement)
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniredshiftparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniredshiftparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
