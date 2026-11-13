package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementWhereDisallowUsingFunctionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &StatementWhereDisallowUsingFunctionAdvisor{})
}

// StatementWhereDisallowUsingFunctionAdvisor is the advisor disallowing
// functions and calculations on indexed columns in WHERE clauses. The
// judgment lives in omni
// (mysql/v1review.StatementWhereDisallowFunctionsAndCalculations); this
// adapter passes the synced schema and renders the findings as advice.
type StatementWhereDisallowUsingFunctionAdvisor struct {
}

func (*StatementWhereDisallowUsingFunctionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mysqlv1review.StatementWhereDisallowFunctionsAndCalculations([]ast.Node{node}, schema) {
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
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}
