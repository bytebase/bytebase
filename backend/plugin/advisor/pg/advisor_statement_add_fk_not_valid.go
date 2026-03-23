package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementAddFKNotValidAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID, &StatementAddFKNotValidAdvisor{})
}

// StatementAddFKNotValidAdvisor is the advisor checking for adding foreign key constraints without NOT VALID.
type StatementAddFKNotValidAdvisor struct {
}

// Check checks for adding foreign key constraints without NOT VALID.
func (*StatementAddFKNotValidAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementAddFKNotValidRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementAddFKNotValidRule struct {
	OmniBaseRule
}

func (*statementAddFKNotValidRule) Name() string {
	return "statement_add_fk_not_valid"
}

func (r *statementAddFKNotValidRule) OnStatement(node ast.Node) {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, cmd := range omniAlterTableCmds(alter) {
		if ast.AlterTableType(cmd.Subtype) != ast.AT_AddConstraint {
			continue
		}
		constraint, ok := cmd.Def.(*ast.Constraint)
		if !ok || constraint.Contype != ast.CONSTR_FOREIGN {
			continue
		}
		if !constraint.SkipValidation {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.StatementAddFKWithValidation.Int32(),
				Title:   r.Title,
				Content: "Adding foreign keys with validation will block reads and writes. You can add check foreign keys not valid and then validate separately",
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	}
}
