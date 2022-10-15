package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
	_ ast.Visitor     = (*insertRowLimitChecker)(nil)
)

func init() {
	advisor.Register(db.Postgres, advisor.PostgreSQLInsertRowLimit, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for to limit INSERT rows.
type InsertRowLimitAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*InsertRowLimitAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmts, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &insertRowLimitChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		maxRow: payload.Number,
	}

	if payload.Number > 0 {
		for _, stmt := range stmts {
			checker.text = stmt.Text()
			checker.line = stmt.LastLine()
			ast.Walk(checker, stmt)
		}
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

type insertRowLimitChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
	maxRow     int
}

// Visit implements the ast.Visitor interface.
func (checker *insertRowLimitChecker) Visit(node ast.Node) ast.Visitor {
	code := advisor.Ok
	rows := 0

	n, ok := node.(*ast.InsertStmt)
	if ok {
		if len(n.ValueList) > checker.maxRow {
			code = advisor.InsertTooManyRows
			rows = len(n.ValueList)
		}
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("The value rows in \"%s\" should be no more than %d, but found %d", checker.text, checker.maxRow, rows),
			Line:    checker.line,
		})
	}
	return checker
}
