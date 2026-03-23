package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &noSelectAllRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type noSelectAllRule struct {
	OmniBaseRule
}

func (*noSelectAllRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL)
}

func (r *noSelectAllRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		r.checkSelectStmt(n, true)
	case *ast.InsertStmt:
		if sel, ok := n.SelectStmt.(*ast.SelectStmt); ok {
			r.checkSelectStmt(sel, false)
		}
	default:
	}
}

func (r *noSelectAllRule) checkSelectStmt(sel *ast.SelectStmt, isTopLevel bool) {
	if sel == nil {
		return
	}

	// For set operations (UNION/INTERSECT/EXCEPT), recurse into children.
	if sel.Op != ast.SETOP_NONE {
		r.checkSelectStmt(sel.Larg, false)
		r.checkSelectStmt(sel.Rarg, false)
		return
	}

	// Walk subqueries first.
	r.walkSelectSubqueries(sel)

	// Check target list for SELECT *.
	if sel.TargetList != nil {
		for _, item := range sel.TargetList.Items {
			rt, ok := item.(*ast.ResTarget)
			if !ok {
				continue
			}
			cr, ok := rt.Val.(*ast.ColumnRef)
			if !ok {
				continue
			}
			if cr.Fields != nil {
				for _, field := range cr.Fields.Items {
					if _, ok := field.(*ast.A_Star); ok {
						stmtText := r.extractSelectText(sel, isTopLevel)
						r.AddAdvice(&storepb.Advice{
							Status:  r.Level,
							Code:    code.StatementSelectAll.Int32(),
							Title:   r.Title,
							Content: fmt.Sprintf("\"%s\" uses SELECT all", stmtText),
							StartPosition: &storepb.Position{
								Line:   r.ContentStartLine(),
								Column: 0,
							},
						})
						return
					}
				}
			}
		}
	}
}

// extractSelectText returns the text representation of a SelectStmt.
func (r *noSelectAllRule) extractSelectText(sel *ast.SelectStmt, isTopLevel bool) string {
	if isTopLevel {
		return r.TrimmedStmtText()
	}
	// For nested subqueries, extract from StmtText using Loc byte offsets.
	loc := sel.Loc
	if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(r.StmtText) {
		return strings.TrimSpace(r.StmtText[loc.Start:loc.End])
	}
	return r.TrimmedStmtText()
}

func (r *noSelectAllRule) walkSelectSubqueries(sel *ast.SelectStmt) {
	if sel == nil {
		return
	}
	r.walkNodeListForSelectAll(sel.FromClause)
	r.walkNodeForSelectAll(sel.WhereClause)
	r.walkNodeForSelectAll(sel.HavingClause)
	r.walkNodeListForSelectAll(sel.GroupClause)
	r.walkNodeListForSelectAll(sel.SortClause)
	r.walkNodeForSelectAll(sel.LimitCount)
	r.walkNodeForSelectAll(sel.LimitOffset)
}

func (r *noSelectAllRule) walkNodeListForSelectAll(list *ast.List) {
	if list == nil {
		return
	}
	for _, item := range list.Items {
		r.walkNodeForSelectAll(item)
	}
}

func (r *noSelectAllRule) walkNodeForSelectAll(node ast.Node) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.SubLink:
		if sub, ok := n.Subselect.(*ast.SelectStmt); ok {
			r.checkSelectStmt(sub, false)
		}
	case *ast.RangeSubselect:
		if sub, ok := n.Subquery.(*ast.SelectStmt); ok {
			r.checkSelectStmt(sub, false)
		}
	case *ast.List:
		r.walkNodeListForSelectAll(n)
	case *ast.ResTarget:
		r.walkNodeForSelectAll(n.Val)
	case *ast.FuncCall:
		r.walkNodeListForSelectAll(n.Args)
	case *ast.A_Expr:
		r.walkNodeForSelectAll(n.Lexpr)
		r.walkNodeForSelectAll(n.Rexpr)
	case *ast.BoolExpr:
		r.walkNodeListForSelectAll(n.Args)
	case *ast.CoalesceExpr:
		r.walkNodeListForSelectAll(n.Args)
	case *ast.CaseExpr:
		r.walkNodeForSelectAll(n.Arg)
		r.walkNodeListForSelectAll(n.Args)
		r.walkNodeForSelectAll(n.Defresult)
	case *ast.CaseWhen:
		r.walkNodeForSelectAll(n.Expr)
		r.walkNodeForSelectAll(n.Result)
	case *ast.NullTest:
		r.walkNodeForSelectAll(n.Arg)
	case *ast.TypeCast:
		r.walkNodeForSelectAll(n.Arg)
	case *ast.JoinExpr:
		r.walkNodeForSelectAll(n.Larg)
		r.walkNodeForSelectAll(n.Rarg)
		r.walkNodeForSelectAll(n.Quals)
	case *ast.SortBy:
		r.walkNodeForSelectAll(n.Node)
	default:
	}
}
