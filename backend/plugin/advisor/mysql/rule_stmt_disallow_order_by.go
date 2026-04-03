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
	_ advisor.Advisor = (*DisallowOrderByAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY, &DisallowOrderByAdvisor{})
}

// DisallowOrderByAdvisor is the advisor checking for no ORDER BY clause in DELETE/UPDATE statements.
type DisallowOrderByAdvisor struct {
}

// Check checks for no ORDER BY clause in DELETE/UPDATE statements.
func (*DisallowOrderByAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &disallowOrderByOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type disallowOrderByOmniRule struct {
	OmniBaseRule
}

func (*disallowOrderByOmniRule) Name() string {
	return "DisallowOrderByRule"
}

func (r *disallowOrderByOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	switch n := node.(type) {
	case *ast.DeleteStmt:
		if len(n.OrderBy) > 0 {
			r.addOrderByAdvice(code.DeleteUseOrderBy, text, n.Loc)
		}
	case *ast.UpdateStmt:
		if len(n.OrderBy) > 0 {
			r.addOrderByAdvice(code.UpdateUseOrderBy, text, n.Loc)
		}
	default:
	}
}

func (r *disallowOrderByOmniRule) addOrderByAdvice(c code.Code, text string, loc ast.Loc) {
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          c.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", text),
		StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(loc))),
	})
}
