package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLNoSelectAll, &NoSelectAllAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLNoSelectAll, &NoSelectAllAdvisor{})
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
	checker := &noSelectAllChecker{level: level}
	for _, stmtNode := range root {
		checker.text = stmtNode.Text()
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type noSelectAllChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	text       string
}

// Enter implements the ast.Visitor interface
func (v *noSelectAllChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.SelectStmt); ok {
		for _, field := range node.Fields.Fields {
			if field.WildCard != nil {
				v.adviceList = append(v.adviceList, advisor.Advice{
					Status:  v.level,
					Code:    common.StatementSelectAll,
					Title:   "Not SELECT all",
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
