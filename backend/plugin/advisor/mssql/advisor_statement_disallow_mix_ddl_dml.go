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
	_ advisor.Advisor = (*StatementDisallowMixDDLDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLStatementDisallowMixDDLDML, &StatementDisallowMixDDLDMLAdvisor{})
}

type StatementDisallowMixDDLDMLAdvisor struct {
}

func (*StatementDisallowMixDDLDMLAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := ctx.Rule.Type

	c := &statementDisallowMixDDLDMLChecker{
		changeType: ctx.ChangeType,
		level:      level,
		title:      title,
	}

	antlr.ParseTreeWalkerDefault.Walk(c, tree)

	if c.hasDDL && c.hasDML {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  level,
			Title:   title,
			Content: "Mixing DDL with DML is not allowed",
			Code:    advisor.StatementDisallowMixDDLDML.Int32(),
		})
	}

	return c.adviceList, nil
}

type statementDisallowMixDDLDMLChecker struct {
	*parser.BaseTSqlParserListener

	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType
	adviceList []*storepb.Advice

	level storepb.Advice_Status
	title string

	hasDML bool
	hasDDL bool
}

func (c *statementDisallowMixDDLDMLChecker) EnterSql_clauses(ctx *parser.Sql_clausesContext) {
	if !tsqlparser.IsTopLevel(ctx.GetParent()) {
		return
	}
	var isDML bool
	if ctx.Dml_clause() != nil {
		isDML = true
		c.hasDML = true
	} else {
		c.hasDDL = true
	}

	switch c.changeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
		if isDML {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Title:   c.title,
				Content: "Alter schema can only run DDL",
				Code:    advisor.StatementDisallowMixDDLDML.Int32(),
				StartPosition: &storepb.Position{
					Line: int32(ctx.GetStart().GetLine()),
				},
			})
		}
	case storepb.PlanCheckRunConfig_DML:
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
}
