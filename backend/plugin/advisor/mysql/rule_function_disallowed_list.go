package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*FunctionDisallowedListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST, &FunctionDisallowedListAdvisor{})
}

// FunctionDisallowedListAdvisor is the advisor checking for disallowed function list.
type FunctionDisallowedListAdvisor struct {
}

func (*FunctionDisallowedListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	var disallowList []string
	for _, fn := range stringArrayPayload.List {
		disallowList = append(disallowList, strings.ToUpper(fn))
	}

	rule := &functionDisallowedListOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		disallowList: disallowList,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type functionDisallowedListOmniRule struct {
	OmniBaseRule
	disallowList []string
}

func (*functionDisallowedListOmniRule) Name() string {
	return "FunctionDisallowedListRule"
}

func (r *functionDisallowedListOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		r.checkSelectStmt(n)
	case *ast.InsertStmt:
		r.checkInsertStmt(n)
	case *ast.UpdateStmt:
		r.checkUpdateStmt(n)
	case *ast.DeleteStmt:
		r.checkDeleteStmt(n)
	case *ast.CreateTableStmt:
		r.checkCreateTableStmt(n)
	case *ast.AlterTableStmt:
		r.checkAlterTableStmt(n)
	default:
	}
}

func (r *functionDisallowedListOmniRule) checkExpr(expr ast.ExprNode) {
	if expr == nil {
		return
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncCallExpr)
		if !ok {
			return true
		}
		if slices.Contains(r.disallowList, strings.ToUpper(fn.Name)) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.DisabledFunction.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Function \"%s\" is disallowed, but \"%s\" uses", fn.Name, r.QueryText()),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(fn.Loc))),
			})
		}
		return true
	})
}

func (r *functionDisallowedListOmniRule) checkCreateTableStmt(n *ast.CreateTableStmt) {
	if n == nil {
		return
	}
	for _, col := range n.Columns {
		if col == nil {
			continue
		}
		r.checkExpr(col.DefaultValue)
		r.checkExpr(col.OnUpdate)
	}
}

func (r *functionDisallowedListOmniRule) checkAlterTableStmt(n *ast.AlterTableStmt) {
	if n == nil {
		return
	}
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		for _, col := range omniGetColumnsFromCmd(cmd) {
			if col == nil {
				continue
			}
			r.checkExpr(col.DefaultValue)
			r.checkExpr(col.OnUpdate)
		}
		r.checkExpr(cmd.DefaultExpr)
	}
}

func (r *functionDisallowedListOmniRule) checkSelectStmt(n *ast.SelectStmt) {
	if n == nil {
		return
	}
	for _, expr := range n.TargetList {
		r.checkExpr(expr)
	}
	r.checkExpr(n.Where)
	for _, expr := range n.GroupBy {
		r.checkExpr(expr)
	}
	r.checkExpr(n.Having)
	for _, item := range n.OrderBy {
		if item != nil {
			r.checkExpr(item.Expr)
		}
	}
}

func (r *functionDisallowedListOmniRule) checkInsertStmt(n *ast.InsertStmt) {
	if n == nil {
		return
	}
	for _, row := range n.Values {
		for _, expr := range row {
			r.checkExpr(expr)
		}
	}
	if n.Select != nil {
		r.checkSelectStmt(n.Select)
	}
}

func (r *functionDisallowedListOmniRule) checkUpdateStmt(n *ast.UpdateStmt) {
	if n == nil {
		return
	}
	for _, assign := range n.SetList {
		if assign != nil {
			r.checkExpr(assign.Value)
		}
	}
	r.checkExpr(n.Where)
}

func (r *functionDisallowedListOmniRule) checkDeleteStmt(n *ast.DeleteStmt) {
	if n == nil {
		return
	}
	r.checkExpr(n.Where)
}
