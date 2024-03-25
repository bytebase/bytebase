package pg

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NonTransactionalAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNonTransactional, &NonTransactionalAdvisor{})
}

type NonTransactionalAdvisor struct {
}

func (*NonTransactionalAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &NonTransactionalChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmt := range stmts {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
		if pg.IsNonTransactionStatement(checker.text) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementNonTransactional,
				Title:   checker.title,
				Content: "This statement is non-transactional",
				Line:    checker.line,
			})
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

type NonTransactionalChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}
