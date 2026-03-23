package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementAddCheckNotValidAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID, &StatementAddCheckNotValidAdvisor{})
}

// StatementAddCheckNotValidAdvisor is the advisor checking for to add check not valid.
type StatementAddCheckNotValidAdvisor struct {
}

// Check checks for to add check not valid.
func (*StatementAddCheckNotValidAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementAddCheckNotValidRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementAddCheckNotValidRule struct {
	OmniBaseRule
}

func (*statementAddCheckNotValidRule) Name() string {
	return "statement_add_check_not_valid"
}

func (r *statementAddCheckNotValidRule) OnStatement(node ast.Node) {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, cmd := range omniAlterTableCmds(alter) {
		if ast.AlterTableType(cmd.Subtype) != ast.AT_AddConstraint {
			continue
		}
		constraint, ok := cmd.Def.(*ast.Constraint)
		if !ok || constraint.Contype != ast.CONSTR_CHECK {
			continue
		}
		if !constraint.SkipValidation {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.StatementAddCheckWithValidation.Int32(),
				Title:   r.Title,
				Content: "Adding check constraints with validation will block reads and writes. You can add check constraints not valid and then validate separately",
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	}
}
