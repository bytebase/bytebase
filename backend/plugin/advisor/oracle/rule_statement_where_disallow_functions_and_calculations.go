// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*StatementWhereDisallowFunctionsAndCalculationsAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &StatementWhereDisallowFunctionsAndCalculationsAdvisor{})
}

// StatementWhereDisallowFunctionsAndCalculationsAdvisor is the Oracle advisor
// that flags functions or calculations applied to indexed columns in WHERE clauses.
// The judgment lives in omni (oracle/v1review.StatementWhereDisallowFunctionsAndCalculations);
// this adapter passes the synced schema and renders the findings as advice.
type StatementWhereDisallowFunctionsAndCalculationsAdvisor struct{}

// Check implements advisor.Advisor.
func (*StatementWhereDisallowFunctionsAndCalculationsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	// Oracle sync stores only the connected owner's objects, under the
	// anonymous schema "".
	schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oraclev1review.StatementWhereDisallowFunctionsAndCalculations([]ast.Node{node}, schema, "", checkCtx.CurrentDatabase) {
			var content string
			switch f.Code {
			case v1review.FunctionOnIndexedColumn:
				content = fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Value, f.Name)
			case v1review.CalculationOnIndexedColumn:
				content = fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Name)
			default:
				continue
			}
			line := stmt.BaseLine() + findingLine(stmt.Text, f)
			if hasWhereFunctionAdvice(adviceList, content, line) {
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: base.ConvertANTLRLineToPosition(line),
			})
		}
	}
	return adviceList, nil
}

// hasWhereFunctionAdvice reports whether an advice with that content is
// already on that line, across statements.
func hasWhereFunctionAdvice(adviceList []*storepb.Advice, content string, line int) bool {
	for _, advice := range adviceList {
		if advice.Content == content && advice.GetStartPosition().GetLine() == int32(line) {
			return true
		}
	}
	return false
}
