package mysql

import (
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

func (*StatementDisallowmixDDLDMLAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	checker := &mysqlparser.StatementTypeChecker{}

	hasDDL := false
	hasDML := false
	for _, stmt := range stmtList {
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
		if checker.IsDDL {
			hasDDL = true
		}
		if checker.IsDML {
			hasDML = true
		}
	}

	if hasDDL && hasDML {
		return []advisor.Advice{
			{
				Status:  level,
				Title:   title,
				Content: "Mixing DDL with DML is not allowed",
				Code:    advisor.StatementDisallowMixDDLDML,
			},
		}, nil
	}

	return []advisor.Advice{
		{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		},
	}, nil
}
