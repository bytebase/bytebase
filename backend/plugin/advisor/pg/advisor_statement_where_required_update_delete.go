package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementWhereRequiredUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE, &StatementWhereRequiredUpdateDeleteAdvisor{})
}

// StatementWhereRequiredUpdateDeleteAdvisor is the advisor checking for WHERE clause requirement in UPDATE/DELETE.
type StatementWhereRequiredUpdateDeleteAdvisor struct {
}

// Check checks for WHERE clause requirement in UPDATE/DELETE statements.
func (*StatementWhereRequiredUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementWhereRequiredUpdateDeleteRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementWhereRequiredUpdateDeleteRule struct {
	OmniBaseRule
}

func (*statementWhereRequiredUpdateDeleteRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE)
}

func (r *statementWhereRequiredUpdateDeleteRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		if n.WhereClause == nil {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.StatementNoWhere.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("\"%s\" requires WHERE clause", r.TrimmedStmtText()),
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	case *ast.DeleteStmt:
		if n.WhereClause == nil {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.StatementNoWhere.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("\"%s\" requires WHERE clause", r.TrimmedStmtText()),
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	default:
	}
}
