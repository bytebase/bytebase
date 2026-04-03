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
	_ advisor.Advisor = (*WhereRequirementForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequirementForSelectAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequirementForSelectAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &WhereRequirementForSelectAdvisor{})
}

// WhereRequirementForSelectAdvisor is the advisor checking for the WHERE clause requirement for SELECT statements.
type WhereRequirementForSelectAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &whereRequirementForSelectOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type whereRequirementForSelectOmniRule struct {
	OmniBaseRule
}

func (*whereRequirementForSelectOmniRule) Name() string {
	return "WhereRequirementForSelectRule"
}

func (r *whereRequirementForSelectOmniRule) OnStatement(node ast.Node) {
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return
	}
	text := strings.TrimSpace(r.StmtText)
	r.checkSelectStmt(sel, text)
}

func (r *whereRequirementForSelectOmniRule) checkSelectStmt(sel *ast.SelectStmt, text string) {
	if sel == nil {
		return
	}

	// Check CTEs.
	for _, cte := range sel.CTEs {
		if cte.Select != nil {
			r.checkSelectStmt(cte.Select, text)
		}
	}

	// Check set operations (UNION left/right).
	if sel.SetOp != ast.SetOpNone {
		r.checkSelectStmt(sel.Left, text)
		r.checkSelectStmt(sel.Right, text)
		return
	}

	// Allow SELECT queries without a FROM clause, e.g. SELECT 1.
	if len(sel.From) == 0 {
		return
	}

	if sel.Where == nil {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.StatementNoWhere.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("\"%s\" requires WHERE clause", text),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(sel.Loc))),
		})
	}

	// Check subqueries in WHERE, FROM, and target list.
	// We inspect each subtree individually (not the whole SelectStmt) to avoid
	// recursing into the SelectStmt itself, which checkSelectStmt already handles.
	var subtrees []ast.Node
	subtrees = append(subtrees, sel.Where)
	for _, from := range sel.From {
		subtrees = append(subtrees, from)
	}
	for _, target := range sel.TargetList {
		subtrees = append(subtrees, target)
	}
	for _, subtree := range subtrees {
		ast.Inspect(subtree, func(n ast.Node) bool {
			if sub, ok := n.(*ast.SelectStmt); ok {
				r.checkSelectStmt(sub, text)
				return false
			}
			return true
		})
	}
}
