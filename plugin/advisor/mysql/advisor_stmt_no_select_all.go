package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(advisor.MySQL, advisor.MySQLNoSelectAll, &NoSelectAllAdvisor{})
	advisor.Register(advisor.TiDB, advisor.MySQLNoSelectAll, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (adv *NoSelectAllAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &noSelectAllChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
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

type noSelectAllChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
}

// Enter implements the ast.Visitor interface
func (v *noSelectAllChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.SelectStmt); ok {
		for _, field := range node.Fields.Fields {
			if field.WildCard != nil {
				v.adviceList = append(v.adviceList, advisor.Advice{
					Status:  v.level,
					Code:    advisor.StatementSelectAll,
					Title:   v.title,
					Content: fmt.Sprintf("\"%s\" uses SELECT all", v.text),
				})
				break
			}
		}

	}
	return in, false
}

// Leave implements the ast.Visitor interface
func (v *noSelectAllChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
