package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
	_ ast.Visitor     = (*noSelectAllChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNoSelectAll, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &noSelectAllChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmt := range stmts {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
		ast.Walk(checker, stmt)
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
	line       int
}

// Visit implements the ast.Visitor interface.
func (checker *noSelectAllChecker) Visit(node ast.Node) ast.Visitor {
	if n, ok := node.(*ast.SelectStmt); ok {
		for _, field := range n.FieldList {
			if column, ok := field.(*ast.ColumnNameDef); ok && column.ColumnName == "*" {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.StatementSelectAll,
					Title:   checker.title,
					Content: fmt.Sprintf("\"%s\" uses SELECT all", checker.text),
					Line:    checker.line,
				})
				break
			}
		}
	}
	return checker
}
