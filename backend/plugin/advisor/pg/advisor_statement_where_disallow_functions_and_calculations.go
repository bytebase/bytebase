// Package pg is the advisor for postgres database.
package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementWhereDisallowFunctionsAndCalculationsAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &StatementWhereDisallowFunctionsAndCalculationsAdvisor{})
}

// StatementWhereDisallowFunctionsAndCalculationsAdvisor is the advisor that flags
// functions or calculations applied to indexed columns in WHERE clauses. The
// judgment lives in omni
// (pg/v1review.StatementWhereDisallowFunctionsAndCalculations); this adapter
// passes the synced schema and its resolved search_path and renders the
// findings as advice.
type StatementWhereDisallowFunctionsAndCalculationsAdvisor struct{}

// Check implements advisor.Advisor.
func (*StatementWhereDisallowFunctionsAndCalculationsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	var schema v1review.Schema
	var searchPath []string
	if checkCtx.DBSchema != nil {
		schema = v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
		// Resolve $user against checkCtx.SessionUser so deployments using
		// `search_path = "$user", public` include the session user's schema.
		searchPath = model.NewDatabaseMetadata(checkCtx.DBSchema, nil, nil,
			storepb.Engine_POSTGRES, checkCtx.IsObjectCaseSensitive).GetSearchPathForCurrentUser(checkCtx.SessionUser)
	}
	if len(searchPath) == 0 {
		// Default PG search_path is `"$user", public`. With no current-user
		// context, fall back to public.
		searchPath = []string{"public"}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		rule := &BaseRule{StmtText: stmt.Text}
		for _, f := range pgv1review.StatementWhereDisallowFunctionsAndCalculations([]ast.Node{node}, schema, searchPath) {
			var content string
			switch f.Code {
			case v1review.FunctionOnIndexedColumn:
				content = fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Value, f.Name)
			case v1review.CalculationOnIndexedColumn:
				content = fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Name)
			default:
				continue
			}
			loc := ast.Loc{Start: f.Start, End: f.End}
			if f.Start == 0 && f.End == 0 {
				loc.Start = -1
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + rule.LocToLine(loc)},
			})
		}
	}
	return adviceList, nil
}
