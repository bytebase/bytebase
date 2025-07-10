package mssql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
}

type StatementDisallowMixInDDLAdvisor struct {
}

func (*StatementDisallowMixInDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
	default:
		return nil, nil
	}
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type

	c := &statementDisallowMixInDDLChecker{
		changeType: checkCtx.ChangeType,
		level:      level,
		title:      title,
	}

	antlr.ParseTreeWalkerDefault.Walk(c, tree)

	return c.adviceList, nil
}

type statementDisallowMixInDDLChecker struct {
	*parser.BaseTSqlParserListener

	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType
	adviceList []*storepb.Advice

	level storepb.Advice_Status
	title string
}

func (c *statementDisallowMixInDDLChecker) EnterSql_clauses(ctx *parser.Sql_clausesContext) {
	if !tsqlparser.IsTopLevel(ctx.GetParent()) {
		return
	}
	var isDML bool
	if ctx.Dml_clause() != nil {
		isDML = true
	}

	if isDML {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Title:         c.title,
			Content:       "Alter schema can only run DDL",
			Code:          advisor.StatementDisallowMixDDLDML.Int32(),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
