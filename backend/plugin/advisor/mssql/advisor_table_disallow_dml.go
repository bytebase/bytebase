// Package mssql is the advisor for MSSQL database.
package mssql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*TableDisallowDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLTableDisallowDML, &TableDisallowDMLAdvisor{})
}

// TableDisallowDMLAdvisor is the advisor checking for disallow DML on specific tables.
type TableDisallowDMLAdvisor struct {
}

func (*TableDisallowDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowDMLChecker{
		level:        level,
		title:        string(checkCtx.Rule.Type),
		disallowList: payload.List,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.adviceList, nil
}

type tableDisallowDMLChecker struct {
	*parser.BaseTSqlParserListener

	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
	// disallowList is the list of table names that disallow DML.
	disallowList []string
}

func (checker *tableDisallowDMLChecker) EnterMerge_statement(ctx *parser.Merge_statementContext) {
	if ctx.Ddl_object() == nil {
		return
	}
	tableName := ctx.Ddl_object().GetText()
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) EnterInsert_statement(ctx *parser.Insert_statementContext) {
	if ctx.Ddl_object() == nil {
		return
	}
	tableName := ctx.Ddl_object().GetText()
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if ctx.Delete_statement_from() == nil {
		return
	}
	tableName := ctx.Delete_statement_from().GetText()
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if ctx.Ddl_object() == nil {
		return
	}
	tableName := ctx.Ddl_object().GetText()
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) EnterSelect_statement_standalone(ctx *parser.Select_statement_standaloneContext) {
	querySpec := ctx.Select_statement().Query_expression().Query_specification()
	if querySpec == nil {
		return
	}
	if querySpec.INTO() == nil || querySpec.Table_name() == nil {
		return
	}
	tableName := tsqlparser.NormalizeTSQLTableName(querySpec.Table_name(), "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	checker.checkTableName(tableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDMLChecker) checkTableName(normalizedTableName string, line int) {
	for _, disallow := range checker.disallowList {
		if normalizedTableName == disallow {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.TableDisallowDML.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("DML is disallowed on table %s.", normalizedTableName),
				StartPosition: common.ConvertANTLRLineToPosition(line),
			})
			return
		}
	}
}
