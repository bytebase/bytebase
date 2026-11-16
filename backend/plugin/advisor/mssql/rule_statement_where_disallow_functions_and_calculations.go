package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &DisallowFuncAndCalculationsAdvisor{})
}

// DisallowFuncAndCalculationsAdvisor is the advisor disallowing functions and
// calculations on indexed columns in WHERE clauses. The judgment lives in omni
// (mssql/v1review.StatementWhereDisallowFunctionsAndCalculations); this
// adapter passes the synced schema and renders the findings as advice.
type DisallowFuncAndCalculationsAdvisor struct{}

var _ advisor.Advisor = (*DisallowFuncAndCalculationsAdvisor)(nil)

func (*DisallowFuncAndCalculationsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var schema v1review.Schema
	if checkCtx.DBSchema != nil {
		schema = v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mssqlv1review.StatementWhereDisallowFunctionsAndCalculations([]ast.Node{node}, schema) {
			var content string
			switch f.Code {
			case v1review.FunctionOnIndexedColumn:
				content = fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Value, f.Name)
			case v1review.CalculationOnIndexedColumn:
				content = fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Name)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: &storepb.Position{Line: findingLine(stmt.Text, f) + int32(stmt.BaseLine())},
			})
		}
	}
	return adviceList, nil
}
