package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
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
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &noSelectAllChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}
	for _, stmt := range stmts {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type noSelectAllChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
}

// Visit implements the ast.Visitor interface.
func (checker *noSelectAllChecker) Visit(node ast.Node) ast.Visitor {
	if n, ok := node.(*ast.SelectStmt); ok {
		for _, field := range n.FieldList {
			if column, ok := field.(*ast.ColumnNameDef); ok && column.ColumnName == "*" {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:        checker.level,
					Code:          advisor.StatementSelectAll.Int32(),
					Title:         checker.title,
					Content:       fmt.Sprintf("\"%s\" uses SELECT all", checker.text),
					StartPosition: common.ConvertPGParserLineToPosition(checker.line),
				})
				break
			}
		}
	}
	return checker
}
