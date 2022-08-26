package mysql

import (
	"github.com/pingcap/tidb/parser/ast"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
	_ ast.Visitor     = (*columnDisallowChangingTypeChecker)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLColumnDisallowChangingType, &ColumnDisallowChangingTypeAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLColumnDisallowChangingType, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type..
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type..
func (*ColumnDisallowChangingTypeAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmtList, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnDisallowChangingTypeChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.OriginTextPosition()
		(stmt).Accept(checker)
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

type columnDisallowChangingTypeChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (*columnDisallowChangingTypeChecker) Enter(in ast.Node) (ast.Node, bool) {
	// TODO: implement it
	// switch node := in.(type) {
	// }

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*columnDisallowChangingTypeChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
