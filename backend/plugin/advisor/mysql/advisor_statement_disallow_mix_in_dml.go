package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
}

type StatementDisallowMixInDMLAdvisor struct {
}

func (*StatementDisallowMixInDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DML:
	default:
		return nil, nil
	}

	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(checkCtx.Rule.Type)

	var adviceList []*storepb.Advice
	for _, stmt := range stmtList {
		checker := &mysqlparser.StatementTypeChecker{}
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)

		if checker.IsDDL {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", checker.Text),
				Code:    advisor.StatementDisallowMixDDLDML.Int32(),
				StartPosition: &storepb.Position{
					Line: int32(stmt.BaseLine) + 1,
				},
			})
		}
	}

	return adviceList, nil
}
