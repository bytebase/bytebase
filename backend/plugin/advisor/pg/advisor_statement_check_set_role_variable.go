package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementCheckSetRoleVariable)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE, &StatementCheckSetRoleVariable{})
}

type StatementCheckSetRoleVariable struct {
}

func (*StatementCheckSetRoleVariable) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementCheckSetRoleVariableRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	advice := RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})

	if !rule.hasSetRole {
		advice = append(advice, &storepb.Advice{
			Status:        level,
			Code:          code.StatementCheckSetRoleVariable.Int32(),
			Title:         rule.Title,
			Content:       "No SET ROLE statement found.",
			StartPosition: nil,
		})
	}

	return advice, nil
}

type statementCheckSetRoleVariableRule struct {
	OmniBaseRule

	hasSetRole      bool
	foundNonSetStmt bool
}

func (*statementCheckSetRoleVariableRule) Name() string {
	return "statement.check-set-role-variable"
}

func (r *statementCheckSetRoleVariableRule) OnStatement(node ast.Node) {
	if setStmt, ok := node.(*ast.VariableSetStmt); ok {
		if !r.foundNonSetStmt && omniIsRoleOrSearchPathSet(setStmt) {
			r.hasSetRole = true
		}
		return
	}

	// Any non-SET statement marks that SET ROLE must have already appeared.
	r.foundNonSetStmt = true
}
