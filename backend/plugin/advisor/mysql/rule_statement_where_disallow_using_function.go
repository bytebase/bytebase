package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementWhereDisallowUsingFunctionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS, &StatementWhereDisallowUsingFunctionAdvisor{})
}

type StatementWhereDisallowUsingFunctionAdvisor struct {
}

func (*StatementWhereDisallowUsingFunctionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &whereDisallowFuncOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type whereDisallowFuncOmniRule struct {
	OmniBaseRule
}

func (*whereDisallowFuncOmniRule) Name() string {
	return "StatementWhereDisallowUsingFunctionRule"
}

func (r *whereDisallowFuncOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	// Collect WHERE expressions from all SelectStmt nodes (including UNION branches).
	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectStmt)
		if !ok {
			return true
		}
		if sel.Where != nil {
			ast.Inspect(sel.Where, func(wn ast.Node) bool {
				if fn, ok := wn.(*ast.FuncCallExpr); ok {
					r.AddAdviceAbsolute(&storepb.Advice{
						Status:        r.Level,
						Code:          code.DisabledFunction.Int32(),
						Title:         r.Title,
						Content:       fmt.Sprintf("Function is disallowed in where clause, but \"%s\" uses", text),
						StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(fn.Loc))),
					})
				}
				return true
			})
		}
		return true
	})
}
