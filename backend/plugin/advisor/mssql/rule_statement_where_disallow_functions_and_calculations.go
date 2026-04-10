package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &DisallowFuncAndCalculationsAdvisor{})
}

type DisallowFuncAndCalculationsAdvisor struct{}

var _ advisor.Advisor = (*DisallowFuncAndCalculationsAdvisor)(nil)

func (*DisallowFuncAndCalculationsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &disallowFuncAndCalcOmniRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type disallowFuncAndCalcOmniRule struct {
	OmniBaseRule
}

func (*disallowFuncAndCalcOmniRule) Name() string {
	return "DisallowFuncAndCalcOmniRule"
}

func (r *disallowFuncAndCalcOmniRule) OnStatement(node ast.Node) {
	// Find all statements that have a WHERE clause.
	ast.Inspect(node, func(n ast.Node) bool {
		var where ast.ExprNode
		switch stmt := n.(type) {
		case *ast.SelectStmt:
			where = stmt.WhereClause
		case *ast.UpdateStmt:
			where = stmt.WhereClause
		case *ast.DeleteStmt:
			where = stmt.WhereClause
		default:
			return true
		}

		if where == nil {
			return true
		}

		r.checkWhereClause(where)
		return true
	})
}

func (r *disallowFuncAndCalcOmniRule) checkWhereClause(where ast.ExprNode) {
	// Only report the first violation per WHERE clause.
	found := false

	ast.Inspect(where, func(n ast.Node) bool {
		if found {
			return false
		}

		switch expr := n.(type) {
		case *ast.FuncCallExpr:
			found = true
			funcName := ""
			if expr.Name != nil {
				funcName = tableRefText(expr.Name)
			}
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Calling function '%s' in 'WHERE' clause is not allowed", funcName),
				StartPosition: &storepb.Position{Line: r.LocToLine(expr.Loc)},
			})
			return false
		case *ast.BinaryExpr:
			if isArithmeticOp(expr.Op) {
				found = true
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
					Title:         r.Title,
					Content:       "Performing calculations in 'WHERE' clause is not allowed",
					StartPosition: &storepb.Position{Line: r.LocToLine(expr.Loc)},
				})
				return false
			}
		case *ast.UnaryExpr:
			if expr.Op == ast.UnaryPlus || expr.Op == ast.UnaryMinus || expr.Op == ast.UnaryBitNot {
				found = true
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
					Title:         r.Title,
					Content:       "Performing calculations in 'WHERE' clause is not allowed",
					StartPosition: &storepb.Position{Line: r.LocToLine(expr.Loc)},
				})
				return false
			}
		default:
		}
		return true
	})
}

func isArithmeticOp(op ast.BinaryOp) bool {
	switch op {
	case ast.BinOpAdd, ast.BinOpSub, ast.BinOpMul, ast.BinOpDiv, ast.BinOpMod:
		return true
	default:
		return false
	}
}
