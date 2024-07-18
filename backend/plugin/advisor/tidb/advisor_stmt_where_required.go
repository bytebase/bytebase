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
func (*WhereRequirementAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
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

	return checker.adviceList, nil
}

type whereRequirementChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
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
		// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
		if node.Where == nil && node.From != nil {
			code = advisor.StatementNoWhere
		}
	}

	if code != advisor.Ok {
		v.adviceList = append(v.adviceList, &storepb.Advice{
			Status:  v.level,
			Code:    code.Int32(),
			Title:   v.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", v.text),
			StartPosition: &storepb.Position{
				Line: int32(v.line),
			},
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*whereRequirementChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
