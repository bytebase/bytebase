package oracle

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
}

type StatementDisallowMixInDMLAdvisor struct {
}

func (*StatementDisallowMixInDMLAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	switch ctx.ChangeType {
	case storepb.PlanCheckRunConfig_DML:
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
			if stmt.Data_manipulation_language_statements() == nil {
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
	}

	return adviceList, nil
}
