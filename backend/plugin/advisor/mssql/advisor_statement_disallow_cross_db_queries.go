package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLStatementDisallowCrossDBQueries, &DisallowCrossDBQueriesAdvisor{})
}

type DisallowCrossDBQueriesAdvisor struct{}

type DisallowCrossDBQueriesChecker struct {
	*parser.BaseTSqlParserListener
	curDB      string
	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
}

func (*DisallowCrossDBQueriesAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &DisallowCrossDBQueriesChecker{
		level: level,
		title: checkCtx.Rule.Type,
		curDB: checkCtx.CurrentDatabase,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.adviceList, nil
}

func (checker *DisallowCrossDBQueriesChecker) EnterTable_source_item(ctx *parser.Table_source_itemContext) {
	if fullTblnameCtx := ctx.Full_table_name(); fullTblnameCtx != nil {
		// Case insensitive.
		if fullTblName, err := tsql.NormalizeFullTableName(fullTblnameCtx); err == nil && fullTblName.Database != "" && !strings.EqualFold(fullTblName.Database, checker.curDB) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    advisor.StatementDisallowCrossDBQueries.Int32(),
				Title:   checker.title,
				Content: fmt.Sprintf("Cross database queries (target databse: '%s', current database: '%s') are prohibited", fullTblName.Database, checker.curDB),
				StartPosition: &storepb.Position{
					Line: int32(ctx.GetStart().GetLine()),
				},
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
