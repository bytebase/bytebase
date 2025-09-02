package pg

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*NonTransactionalAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementNonTransactional, &NonTransactionalAdvisor{})
}

type NonTransactionalAdvisor struct {
}

func (*NonTransactionalAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &NonTransactionalChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}
	for _, stmt := range stmts {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
		if pg.IsNonTransactionStatement(checker.text) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.StatementNonTransactional.Int32(),
				Title:         checker.title,
				Content:       "This statement is non-transactional",
				StartPosition: common.ConvertPGParserLineToPosition(checker.line),
			})
		}
	}

	return checker.adviceList, nil
}

type NonTransactionalChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
}
