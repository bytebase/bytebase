package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*WhereRequirementForUpdateDeleteAdvisor)(nil)
	_ ast.Visitor     = (*whereRequirementForUpdateDeleteChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLWhereRequirementForUpdateDelete, &WhereRequirementForUpdateDeleteAdvisor{})
}

// WhereRequirementForUpdateDeleteAdvisor is the advisor checking for the WHERE clause requirement.
type WhereRequirementForUpdateDeleteAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementForUpdateDeleteAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &whereRequirementForUpdateDeleteChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmts {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type whereRequirementForUpdateDeleteChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
}

// Visit implements the ast.Visitor interface.
func (checker *whereRequirementForUpdateDeleteChecker) Visit(node ast.Node) ast.Visitor {
	code := advisor.Ok
	switch n := node.(type) {
	// DELETE
	case *ast.DeleteStmt:
		if n.WhereClause == nil {
			code = advisor.StatementNoWhere
		}
	// UPDATE
	case *ast.UpdateStmt:
		if n.WhereClause == nil {
			code = advisor.StatementNoWhere
		}
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  checker.level,
			Code:    code.Int32(),
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" requires WHERE clause", checker.text),
			StartPosition: &storepb.Position{
				Line: int32(checker.line),
			},
		})
	}
	return checker
}
