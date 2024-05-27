package mssql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(store.Engine_MSSQL, advisor.MSSQLStatementDisallowCrossDBQueries, &DisallowCrossDBQueriesAdvisor{})
}

type DisallowCrossDBQueriesAdvisor struct{}

type DisallowCrossDBQueriesChecker struct {
	*parser.BaseTSqlParserListener
	curDB      string
	level      advisor.Status
	title      string
	adviceList []advisor.Advice
}

func (*DisallowCrossDBQueriesAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &DisallowCrossDBQueriesChecker{
		level: level,
		title: ctx.Rule.Type,
		curDB: ctx.CurrentDatabase,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

func (checker *DisallowCrossDBQueriesChecker) EnterTable_source_item(ctx *parser.Table_source_itemContext) {
	if fullTblnameCtx := ctx.Full_table_name(); fullTblnameCtx != nil {
		// Case insensitive.
		if fullTblName, err := tsql.NormalizeFullTableName(fullTblnameCtx); err == nil && fullTblName.Database != "" && !strings.EqualFold(fullTblName.Database, checker.curDB) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.StatementDisallowCrossDBQueries,
				Title:   checker.title,
				Content: fmt.Sprintf("Cross database queries (target databse: '%s', current database: '%s') are prohibited", fullTblName.Database, checker.curDB),
				Line:    ctx.GetStart().GetLine(),
			})
		}
		// Ignore internal error...
	}
}

func (checker *DisallowCrossDBQueriesChecker) EnterUse_statement(ctx *parser.Use_statementContext) {
	if newDB := ctx.GetDatabase(); newDB != nil {
		_, lowercaceDBName := tsql.NormalizeTSQLIdentifier(newDB)
		checker.curDB = lowercaceDBName
	}
}
