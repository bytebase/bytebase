package tidb

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/pingcap/tidb/parser/ast"
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
		(stmtNode).Accept(checker)
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
	// only used by select statement.
	// We use selectDownwards as the access direction flag for traversing ast to identify leaf nodes.
	// after accessing a leaf node, we change selectDownwards to false to change the access direction.
	selectDownwards bool
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
		v.selectDownwards = true
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
func (v *whereRequirementChecker) Leave(in ast.Node) (ast.Node, bool) {
	if !v.selectDownwards {
		return in, true
	}
	if node, ok := in.(*ast.SelectStmt); ok {
		v.selectDownwards = false
		if node.Where == nil {
			v.adviceList = append(v.adviceList, advisor.Advice{
				Status:  v.level,
				Code:    advisor.StatementNoWhere,
				Title:   v.title,
				Content: fmt.Sprintf("\"%s\" requires WHERE clause", v.text),
				Line:    v.line,
			})
		}
	}
	return in, true
}
