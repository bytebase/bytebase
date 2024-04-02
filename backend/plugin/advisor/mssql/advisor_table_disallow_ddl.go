// Package mssql is the advisor for MSSQL database.
package mssql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableDisallowDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLTableDisallowDDL, &TableDisallowDDLAdvisor{})
}

// TableDisallowDDLAdvisor is the advisor checking for disallow DDL on specific tables.
type TableDisallowDDLAdvisor struct {
}

func (*TableDisallowDDLAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowDDLChecker{
		level:        level,
		title:        string(ctx.Rule.Type),
		disallowList: payload.List,
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

type tableDisallowDDLChecker struct {
	*parser.BaseTSqlParserListener

	level      advisor.Status
	title      string
	adviceList []advisor.Advice
	// disallowList is the list of table names that disallow DDL.
	disallowList []string
}

func (checker *tableDisallowDDLChecker) EnterCreate_table(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	checker.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDDLChecker) EnterAlter_table(ctx *parser.Alter_tableContext) {
	tableName := ctx.Table_name(0)
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	checker.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDDLChecker) EnterDrop_table(ctx *parser.Drop_tableContext) {
	for _, tableName := range ctx.AllTable_name() {
		if tableName == nil {
			return
		}
		normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
		checker.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
	}
}

func (checker *tableDisallowDDLChecker) EnterTruncate_table(ctx *parser.Truncate_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "" /* fallbackSchema */, false /* caseSensitive */)
	checker.checkTableName(normalizedTableName, ctx.GetStart().GetLine())
}

func (checker *tableDisallowDDLChecker) checkTableName(normalizedTableName string, line int) {
	for _, disallow := range checker.disallowList {
		if normalizedTableName == disallow {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.TableDisallowDDL,
				Title:   checker.title,
				Content: fmt.Sprintf("DDL is disallowed on table %s.", normalizedTableName),
				Line:    line,
			})
			return
		}
	}
}
