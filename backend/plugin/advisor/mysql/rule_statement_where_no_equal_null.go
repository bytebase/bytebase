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
	_ advisor.Advisor = (*StatementWhereNoEqualNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL, &StatementWhereNoEqualNullAdvisor{})
}

type StatementWhereNoEqualNullAdvisor struct {
}

func (*StatementWhereNoEqualNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &whereNoEqualNullOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type whereNoEqualNullOmniRule struct {
	OmniBaseRule
}

func (*whereNoEqualNullOmniRule) Name() string {
	return "StatementWhereNoEqualNullRule"
}

func (r *whereNoEqualNullOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	// Collect WHERE expressions from all SelectStmt nodes (including UNION branches).
	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectStmt)
		if !ok {
			return true
		}
		if sel.Where != nil {
			ast.Inspect(sel.Where, func(wn ast.Node) bool {
				if bin, ok := wn.(*ast.BinaryExpr); ok {
					if (bin.Op == ast.BinOpEq || bin.Op == ast.BinOpNe) && isNullLiteral(bin.Right) {
						r.AddAdviceAbsolute(&storepb.Advice{
							Status:        r.Level,
							Code:          code.StatementWhereNoEqualNull.Int32(),
							Title:         r.Title,
							Content:       fmt.Sprintf("WHERE clause contains equal null: %s", text),
							StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(bin.Loc))),
						})
					}
				}
				return true
			})
		}
		return true
	})
}

func isNullLiteral(expr ast.ExprNode) bool {
	_, ok := expr.(*ast.NullLit)
	return ok
}
