package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db/pg"
)

var (
	_ advisor.Advisor = (*NonTransactionalAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL, &NonTransactionalAdvisor{})
}

// NonTransactionalAdvisor is the advisor checking for non-transactional statements.
type NonTransactionalAdvisor struct {
}

// Check checks for non-transactional statements.
func (*NonTransactionalAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &nonTransactionalRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type nonTransactionalRule struct {
	OmniBaseRule
}

func (*nonTransactionalRule) Name() string {
	return "statement.non-transactional"
}

func (r *nonTransactionalRule) OnStatement(node ast.Node) {
	switch node.(type) {
	case *ast.DropdbStmt, *ast.IndexStmt, *ast.DropStmt, *ast.VacuumStmt:
	default:
		return
	}

	stmtText := r.TrimmedStmtText()
	if pg.IsNonTransactionStatement(stmtText) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.StatementNonTransactional.Int32(),
			Title:   r.Title,
			Content: "This statement is non-transactional",
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
