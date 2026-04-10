package mssql

import (
	"context"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE, &ProcedureDisallowCreateOrAlterAdvisor{})
}

type ProcedureDisallowCreateOrAlterAdvisor struct{}

func (*ProcedureDisallowCreateOrAlterAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	rule := &procedureDisallowCreateOrAlterRule{OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()}}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type procedureDisallowCreateOrAlterRule struct {
	OmniBaseRule
}

func (*procedureDisallowCreateOrAlterRule) Name() string {
	return "ProcedureDisallowCreateOrAlterRule"
}

func (r *procedureDisallowCreateOrAlterRule) OnStatement(node ast.Node) {
	if n, ok := node.(*ast.CreateProcedureStmt); ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DisallowCreateProcedure.Int32(),
			Title:         r.Title,
			Content:       "Creating or altering procedures is prohibited",
			StartPosition: &storepb.Position{Line: r.LocToLine(n.Loc)},
		})
	}
}
