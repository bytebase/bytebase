package oracle

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
}

type StatementDisallowMixInDDLAdvisor struct {
}

func (*StatementDisallowMixInDDLAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	switch ctx.ChangeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
	default:
		return nil, nil
	}
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
	for _, item := range tree.GetChildren() {
		if stmt, ok := item.(parser.IUnit_statementContext); ok {
			if stmt.Data_manipulation_language_statements() != nil {
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
		}
	}

	return adviceList, nil
}
