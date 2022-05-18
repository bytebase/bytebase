package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"

	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*WhereRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
}

// WhereRequirementAdvisor is the advisor checking for the WHERE clause requirement.
type WhereRequirementAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (adv *WhereRequirementAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	we := &whereRequirementChecker{level: level}
	for _, stmtNode := range root {
		(stmtNode).Accept(we)
	}

	if len(we.adviceList) == 0 {
		we.adviceList = append(we.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return we.adviceList, nil
}

type whereRequirementChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
}

// Enter implements the ast.Visitor interface
func (v *whereRequirementChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := common.Ok
	switch node := in.(type) {
	// DELETE
	case *ast.DeleteStmt:
		if node.Where == nil {
			code = common.StatementNoWhere
		}
	// UPDATE
	case *ast.UpdateStmt:
		if node.Where == nil {
			code = common.StatementNoWhere
		}
	// SELECT
	case *ast.SelectStmt:
		if node.Where == nil {
			code = common.StatementNoWhere
		}
	}

	if code != common.Ok {
		v.adviceList = append(v.adviceList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   "Require WHERE clause",
			Content: fmt.Sprintf("%q requires WHERE clause", in.Text()),
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface
func (v *whereRequirementChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
