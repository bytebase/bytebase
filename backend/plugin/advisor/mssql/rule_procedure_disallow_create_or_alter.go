package mssql

import (
	"context"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE, &ProcedureDisallowCreateOrAlterAdvisor{})
}

// ProcedureDisallowCreateOrAlterAdvisor is the advisor disallowing creating
// or altering procedures. The judgment lives in omni
// (mssql/v1review.SystemProcedureDisallowCreate); this adapter renders the
// findings as advice.
type ProcedureDisallowCreateOrAlterAdvisor struct{}

func (*ProcedureDisallowCreateOrAlterAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mssqlv1review.SystemProcedureDisallowCreate([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.DisallowCreateProcedure.Int32(),
				Title:         title,
				Content:       "Creating or altering procedures is prohibited",
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
			})
		}
	}
	return adviceList, nil
}
