package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementWhereMaximumLogicalOperatorCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT, &StatementWhereMaximumLogicalOperatorCountAdvisor{})
}

type StatementWhereMaximumLogicalOperatorCountAdvisor struct {
}

func (*StatementWhereMaximumLogicalOperatorCountAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	maximum := int(numberPayload.Number)
	var allAdvice []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		rule := &maxLogicalOperatorOmniRule{
			OmniBaseRule: OmniBaseRule{
				Level: level,
				Title: checkCtx.Rule.Type.String(),
			},
			maximum: maximum,
		}
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		if br, ok2 := any(rule).(interface{ SetStatement(int, string) }); ok2 {
			br.SetStatement(stmt.BaseLine(), stmt.Text)
		}
		rule.OnStatement(node)
		allAdvice = append(allAdvice, rule.GetAdviceList()...)
	}

	return allAdvice, nil
}

type maxLogicalOperatorOmniRule struct {
	OmniBaseRule
	maximum int
}

func (*maxLogicalOperatorOmniRule) Name() string {
	return "StatementWhereMaximumLogicalOperatorCountRule"
}

func (r *maxLogicalOperatorOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	r.walkNode(node, text)
}

func (r *maxLogicalOperatorOmniRule) walkNode(node ast.Node, text string) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		r.checkSelect(n, text)
	case *ast.UpdateStmt:
		r.checkExpr(n.Where, text)
	case *ast.DeleteStmt:
		r.checkExpr(n.Where, text)
	case *ast.InsertStmt:
		if n.Select != nil {
			r.checkSelect(n.Select, text)
		}
	default:
	}
}

func (r *maxLogicalOperatorOmniRule) checkSelect(sel *ast.SelectStmt, text string) {
	if sel == nil {
		return
	}
	if sel.SetOp != ast.SetOpNone {
		r.checkSelect(sel.Left, text)
		r.checkSelect(sel.Right, text)
		return
	}
	r.checkExpr(sel.Where, text)
}

func (r *maxLogicalOperatorOmniRule) checkExpr(expr ast.ExprNode, text string) {
	if expr == nil {
		return
	}
	// Count OR operands.
	orCount := r.countOrOperands(expr)
	if orCount > r.maximum {
		line := r.findFirstOrLine(expr)
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.StatementWhereMaximumLogicalOperatorCount.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Number of tokens (%d) in the OR predicate operation exceeds limit (%d) in statement %q.", orCount, r.maximum, text),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(line)),
		})
	}
	// Check IN lists.
	r.walkExprForIn(expr, text)
}

// countOrOperands returns the number of operands in an OR chain.
// For `a OR b OR c OR d OR e` (4 OR operators), returns 5.
func (*maxLogicalOperatorOmniRule) countOrOperands(expr ast.ExprNode) int {
	if expr == nil {
		return 0
	}
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != ast.BinOpOr {
		return 0
	}
	count := 0
	for {
		count++
		left, ok := bin.Left.(*ast.BinaryExpr)
		if !ok || left.Op != ast.BinOpOr {
			count++ // count the final left operand
			break
		}
		bin = left
	}
	return count
}

func (r *maxLogicalOperatorOmniRule) findFirstOrLine(expr ast.ExprNode) int32 {
	if expr == nil {
		return r.ContentStartLine()
	}
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != ast.BinOpOr {
		return r.ContentStartLine()
	}
	// Find the deepest OR.
	if left, ok := bin.Left.(*ast.BinaryExpr); ok && left.Op == ast.BinOpOr {
		return r.findFirstOrLine(bin.Left)
	}
	return r.LocToLine(bin.Loc)
}

func (r *maxLogicalOperatorOmniRule) walkExprForIn(expr ast.ExprNode, text string) {
	ast.Inspect(expr, func(n ast.Node) bool {
		e, ok := n.(*ast.InExpr)
		if !ok {
			return true
		}
		count := len(e.List)
		if count > r.maximum {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementWhereMaximumLogicalOperatorCount.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Number of tokens (%d) in IN predicate operation exceeds limit (%d) in statement %q.", count, r.maximum, text),
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(e.Loc))),
			})
		}
		return false
	})
}
