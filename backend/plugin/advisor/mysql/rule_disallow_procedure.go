package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ProcedureDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE, &ProcedureDisallowCreateAdvisor{})
}

// ProcedureDisallowCreateAdvisor is the advisor checking for disallow create procedure.
type ProcedureDisallowCreateAdvisor struct {
}

// Check checks for disallow create procedure.
func (*ProcedureDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &procedureDisallowCreateOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type procedureDisallowCreateOmniRule struct {
	OmniBaseRule
}

func (*procedureDisallowCreateOmniRule) Name() string {
	return "ProcedureDisallowCreateRule"
}

func (r *procedureDisallowCreateOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreateFunctionStmt)
	if !ok || !n.IsProcedure {
		return
	}
	if n.Name != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          advisorcode.DisallowCreateProcedure.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("Procedure is forbidden, but \"%s\" creates", r.QueryText()),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
		})
	}
}
