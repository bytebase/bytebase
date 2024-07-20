package oracle

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowmixDDLDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleStatementDisallowMixDDLDML, &StatementDisallowmixDDLDMLAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleStatementDisallowMixDDLDML, &StatementDisallowmixDDLDMLAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleStatementDisallowMixDDLDML, &StatementDisallowmixDDLDMLAdvisor{})
}

type StatementDisallowmixDDLDMLAdvisor struct {
}

func (*StatementDisallowmixDDLDMLAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := ctx.Rule.Type

	var adviceList []*storepb.Advice
	var hasDDL, hasDML bool
	for _, item := range tree.GetChildren() {
		if stmt, ok := item.(parser.IUnit_statementContext); ok {
			var isDDL, isDML bool
			if stmt.Data_manipulation_language_statements() != nil {
				isDML = true
			} else {
				isDDL = true
			}

			switch ctx.ChangeType {
			case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
				if isDML {
					adviceList = append(adviceList, &storepb.Advice{
						Status:  level,
						Title:   title,
						Content: "Alter schema can only run DDL",
						Code:    advisor.StatementDisallowMixDDLDML.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.GetStart().GetLine()),
						},
					})
				}
			case storepb.PlanCheckRunConfig_DML:
				if isDDL {
					adviceList = append(adviceList, &storepb.Advice{
						Status:  level,
						Title:   title,
						Content: "Data change can only run DML",
						Code:    advisor.StatementDisallowMixDDLDML.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.GetStart().GetLine()),
						},
					})
				}
			}

			if isDDL {
				hasDDL = true
			}
			if isDML {
				hasDML = true
			}
		}
	}

	if hasDDL && hasDML {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: "Mixing DDL with DML is not allowed",
			Code:    advisor.StatementDisallowMixDDLDML.Int32(),
		})
	}

	return adviceList, nil
}
