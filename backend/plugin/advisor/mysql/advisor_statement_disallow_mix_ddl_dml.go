package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowmixDDLDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementDisallowMixDDLDML, &StatementDisallowmixDDLDMLAdvisor{})
}

type StatementDisallowmixDDLDMLAdvisor struct {
}

func (*StatementDisallowmixDDLDMLAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	var adviceList []*storepb.Advice
	var hasDDL, hasDML bool
	for _, stmt := range stmtList {
		checker := &mysqlparser.StatementTypeChecker{}
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)

		switch ctx.ChangeType {
		case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
			if checker.IsDML {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Alter schema can only run DDL, \"%s\" is not DDL", checker.Text),
					Code:    advisor.StatementDisallowMixDDLDML.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.BaseLine),
					},
				})
			}
		case storepb.PlanCheckRunConfig_DML:
			if checker.IsDDL {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", checker.Text),
					Code:    advisor.StatementDisallowMixDDLDML.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.BaseLine),
					},
				})
			}
		}

		if checker.IsDDL {
			hasDDL = true
		}
		if checker.IsDML {
			hasDML = true
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
