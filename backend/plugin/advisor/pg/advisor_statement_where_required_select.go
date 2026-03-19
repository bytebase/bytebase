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
	_ advisor.Advisor = (*StatementWhereRequiredSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &StatementWhereRequiredSelectAdvisor{})
}

// StatementWhereRequiredSelectAdvisor is the advisor checking for WHERE clause requirement in SELECT statements.
type StatementWhereRequiredSelectAdvisor struct {
}

// Check checks for WHERE clause requirement in SELECT statements.
func (*StatementWhereRequiredSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementWhereRequiredSelectRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementWhereRequiredSelectRule struct {
	OmniBaseRule
}

func (*statementWhereRequiredSelectRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT)
}

func (r *statementWhereRequiredSelectRule) OnStatement(node ast.Node) {
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return
	}
	r.checkSelectStmt(sel, true)
}

// checkSelectStmt recursively checks a SelectStmt for WHERE clause requirement.
// isTopLevel indicates whether this is the top-level statement (for text extraction).
func (r *statementWhereRequiredSelectRule) checkSelectStmt(sel *ast.SelectStmt, isTopLevel bool) {
	if sel == nil {
		return
	}

	// For set operations (UNION/INTERSECT/EXCEPT), recurse into children.
	if sel.Op != ast.SETOP_NONE {
		r.checkSelectStmt(sel.Larg, false)
		r.checkSelectStmt(sel.Rarg, false)
		return
	}

	// Walk into subqueries in all expressions before checking this SELECT.
	r.walkSubqueries(sel)

	// Skip SELECTs without FROM clause (e.g. SELECT 1).
	if sel.FromClause == nil || len(sel.FromClause.Items) == 0 {
		return
	}

	// If there's a WHERE clause, it's fine.
	if sel.WhereClause != nil {
		return
	}

	stmtText := r.extractSelectText(sel, isTopLevel)

	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.StatementNoWhere.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}

// extractSelectText returns the text representation of a SelectStmt.
func (r *statementWhereRequiredSelectRule) extractSelectText(sel *ast.SelectStmt, isTopLevel bool) string {
	if isTopLevel {
		return strings.TrimSpace(r.StmtText)
	}
	// For nested subqueries, extract from StmtText using Loc byte offsets.
	loc := sel.Loc
	if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(r.StmtText) {
		start := loc.Start
		end := loc.End
		// Include surrounding parentheses if present.
		if start > 0 && r.StmtText[start-1] == '(' && end < len(r.StmtText) && r.StmtText[end] == ')' {
			start--
			end++
		}
		return r.StmtText[start:end]
	}
	return strings.TrimSpace(r.StmtText)
}

// walkSubqueries walks the AST nodes within a SelectStmt to find nested subqueries.
func (r *statementWhereRequiredSelectRule) walkSubqueries(sel *ast.SelectStmt) {
	// Check target list expressions for subqueries.
	r.walkNodeList(sel.TargetList)
	// Check FROM clause for subqueries (e.g., subqueries in FROM).
	r.walkNodeList(sel.FromClause)
	// Check WHERE clause for subqueries.
	r.walkNode(sel.WhereClause)
	// Check HAVING clause for subqueries.
	r.walkNode(sel.HavingClause)
	// Check GROUP BY clause.
	r.walkNodeList(sel.GroupClause)
	// Check sort clause.
	r.walkNodeList(sel.SortClause)
	// Check limit/offset.
	r.walkNode(sel.LimitCount)
	r.walkNode(sel.LimitOffset)
}

func (r *statementWhereRequiredSelectRule) walkNodeList(list *ast.List) {
	if list == nil {
		return
	}
	for _, item := range list.Items {
		r.walkNode(item)
	}
}

func (r *statementWhereRequiredSelectRule) walkNode(node ast.Node) {
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
		r.walkNodeList(n)
	case *ast.ResTarget:
		r.walkNode(n.Val)
	case *ast.FuncCall:
		r.walkNodeList(n.Args)
	case *ast.A_Expr:
		r.walkNode(n.Lexpr)
		r.walkNode(n.Rexpr)
	case *ast.BoolExpr:
		r.walkNodeList(n.Args)
	case *ast.CoalesceExpr:
		r.walkNodeList(n.Args)
	case *ast.CaseExpr:
		r.walkNode(n.Arg)
		r.walkNodeList(n.Args)
		r.walkNode(n.Defresult)
	case *ast.CaseWhen:
		r.walkNode(n.Expr)
		r.walkNode(n.Result)
	case *ast.NullTest:
		r.walkNode(n.Arg)
	case *ast.TypeCast:
		r.walkNode(n.Arg)
	case *ast.JoinExpr:
		r.walkNode(n.Larg)
		r.walkNode(n.Rarg)
		r.walkNode(n.Quals)
	case *ast.SortBy:
		r.walkNode(n.Node)
	default:
	}
}
