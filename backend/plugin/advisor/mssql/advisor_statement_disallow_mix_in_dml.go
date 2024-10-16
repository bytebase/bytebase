package mssql

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
}

type StatementDisallowMixInDMLAdvisor struct {
}

func (*StatementDisallowMixInDMLAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
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

	c := &statementDisallowMixInDMLChecker{
		changeType: ctx.ChangeType,
		level:      level,
		title:      title,
	}

	antlr.ParseTreeWalkerDefault.Walk(c, tree)

	return c.adviceList, nil
}

type statementDisallowMixInDMLChecker struct {
	*parser.BaseTSqlParserListener

	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType
	adviceList []*storepb.Advice

	level storepb.Advice_Status
	title string
}

func (c *statementDisallowMixInDMLChecker) EnterSql_clauses(ctx *parser.Sql_clausesContext) {
	if !tsqlparser.IsTopLevel(ctx.GetParent()) {
		return
	}
	var isDML bool
	if ctx.Dml_clause() != nil {
		isDML = true
	}

	if !isDML {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Title:   c.title,
			Content: "Data change can only run DML",
			Code:    advisor.StatementDisallowMixDDLDML.Int32(),
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
}
