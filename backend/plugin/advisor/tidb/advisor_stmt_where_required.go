package tidb

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

var (
	_ advisor.Advisor = (*WhereRequirementAdvisor)(nil)
	_ ast.Visitor     = (*whereRequirementChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
}

// WhereRequirementAdvisor is the advisor checking for the WHERE clause requirement.
type WhereRequirementAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &whereRequirementChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		checker.line = stmtNode.OriginTextPosition()
		// (stmtNode).Accept(checker)
		checker.checkSelect(stmtNode)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type whereRequirementChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int

	cteOuterSchema []string
}

// Enter implements the ast.Visitor interface.
func (v *whereRequirementChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisor.Ok
	switch node := in.(type) {
	// DELETE
	case *ast.DeleteStmt:
		if node.Where == nil {
			code = advisor.StatementNoWhere
		}
	// UPDATE
	case *ast.UpdateStmt:
		if node.Where == nil {
			code = advisor.StatementNoWhere
		}
	// SELECT
	case *ast.SelectStmt:
		if node.Where == nil {
			code = advisor.StatementNoWhere
		}
	}

	if code != advisor.Ok {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   v.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", v.text),
			Line:    v.line,
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*whereRequirementChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func (v *whereRequirementChecker) checkSelect(in ast.Node) {
	switch in.(type) {
	case *ast.SelectStmt:
	case *ast.SetOprStmt:
	default:
		// only focus on select statement or set statement.
		return
	}
	v.checkNode(in)
}

func (v *whereRequirementChecker) checkNode(in ast.Node) bool {
	switch node := in.(type) {
	case *ast.SelectStmt:
		return v.checkSelectStmt(node)
	case *ast.Join:
		return v.checkJoin(node)
	case *ast.TableSource:
		return v.checkTableSource(node)
	case *ast.SetOprStmt:
		return v.checkSetOpr(node)
	}
	return true
}

func (v *whereRequirementChecker) checkSetOpr(node *ast.SetOprStmt) bool {
	if node.With != nil {
		cteOuterLength := len(v.cteOuterSchema)
		defer func() {
			v.cteOuterSchema = v.cteOuterSchema[:cteOuterLength]
		}()
		for _, cte := range node.With.CTEs {
			v.checkCTE(cte)
			v.cteOuterSchema = append(v.cteOuterSchema, cte.Name.O)
		}
	}

	for _, selectStmt := range node.SelectList.Selects {
		v.checkNode(selectStmt)
	}
	return false
}

func (v *whereRequirementChecker) checkRecursiveCTE(node *ast.CommonTableExpression) bool {
	switch x := node.Query.Query.(type) {
	case *ast.SetOprStmt:
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				v.checkCTE(cte)
			}
		}
	default:
		v.checkNonRecursiveCTE(node)
	}
	return false
}

func (v *whereRequirementChecker) checkNonRecursiveCTE(node *ast.CommonTableExpression) bool {
	return v.checkNode(node.Query.Query)
}

func (v *whereRequirementChecker) checkCTE(node *ast.CommonTableExpression) bool {
	if node.IsRecursive {
		return v.checkRecursiveCTE(node)
	}
	return v.checkNonRecursiveCTE(node)
}

func (v *whereRequirementChecker) checkSelectStmt(node *ast.SelectStmt) bool {
	if node.With != nil {
		cteOuterLength := len(v.cteOuterSchema)
		defer func() {
			v.cteOuterSchema = v.cteOuterSchema[:cteOuterLength]
		}()
		for _, cte := range node.With.CTEs {
			v.checkCTE(cte)
			v.cteOuterSchema = append(v.cteOuterSchema, cte.Name.O)
		}
	}

	isPhysical := false
	if node.From != nil {
		isPhysical = v.checkNode(node.From.TableRefs)
	}

	if isPhysical && node.Where == nil {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  v.level,
			Code:    advisor.StatementNoWhere,
			Title:   v.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", v.text),
			Line:    v.line,
		})
	}

	if node.Where != nil {
		v.checkExprNode(node.Where)
	}

	if node.Fields != nil {
		for _, field := range node.Fields.Fields {
			if field.WildCard != nil {
				continue
			}

			v.checkExprNode(field.Expr)
		}
	}

	return false
}

func (v *whereRequirementChecker) checkTableSource(in *ast.TableSource) bool {
	switch node := in.Source.(type) {
	case *ast.TableName:
		// search whether this table name is a cte name.
		for i := len(v.cteOuterSchema) - 1; i >= 0; i-- {
			if node.Name.O == v.cteOuterSchema[i] {
				return false
			}
		}
		return true
	default:
		v.checkNode(in.Source)
		return false
	}
}

func (v *whereRequirementChecker) checkJoin(node *ast.Join) bool {
	if node.Right == nil {
		return v.checkNode(node.Left)
	}

	// check left
	v.checkNode(node.Left)
	v.checkNode(node.Right)
	return false
}

func (v *whereRequirementChecker) checkExprNode(in ast.ExprNode) bool {
	if in == nil {
		return false
	}

	switch node := in.(type) {
	case *ast.BinaryOperationExpr:
		v.checkExprNode(node.L)
		v.checkExprNode(node.R)
		return false
	case *ast.AggregateFuncExpr:
		v.checkExprNodeList(node.Args)
		return false
	case *ast.SubqueryExpr:
		v.checkNode(node.Query)
		return false
	}
	return false
}

func (v *whereRequirementChecker) checkExprNodeList(nodeList []ast.ExprNode) bool {
	for _, node := range nodeList {
		v.checkExprNode(node)
	}
	return false
}
