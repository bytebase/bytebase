package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowRemoveTblCascadeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE, &StatementDisallowRemoveTblCascadeAdvisor{})
}

// StatementDisallowRemoveTblCascadeAdvisor is the advisor checking for disallow CASCADE option when removing tables.
type StatementDisallowRemoveTblCascadeAdvisor struct {
}

// Check checks for CASCADE option in DROP TABLE and TRUNCATE TABLE statements.
func (*StatementDisallowRemoveTblCascadeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowRemoveTblCascadeRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementDisallowRemoveTblCascadeRule struct {
	OmniBaseRule
}

func (*statementDisallowRemoveTblCascadeRule) Name() string {
	return "statement.disallow-remove-table-cascade"
}

func (r *statementDisallowRemoveTblCascadeRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.DropStmt:
		if n.RemoveType == int(ast.OBJECT_TABLE) && ast.DropBehavior(n.Behavior) == ast.DROP_CASCADE {
			r.addCascadeAdvice()
		}
	case *ast.TruncateStmt:
		if n.Behavior == ast.DROP_CASCADE {
			r.addCascadeAdvice()
		}
	default:
	}
}

func (r *statementDisallowRemoveTblCascadeRule) addCascadeAdvice() {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.StatementDisallowCascade.Int32(),
		Title:   r.Title,
		Content: "The use of CASCADE is not permitted when removing a table",
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
