package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableRequireCollationAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableRequireCollation, &TableRequireCollationAdvisor{})
}

// TableRequireCollationAdvisor is the advisor checking for require collation.
type TableRequireCollationAdvisor struct {
}

func (*TableRequireCollationAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableRequireCollationChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type tableRequireCollationChecker struct {
	*mysql.BaseMySQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	baseLine   int
}

func (checker *tableRequireCollationChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}

	hasCollation := false
	if ctx.CreateTableOptions() != nil {
		for _, tableOption := range ctx.CreateTableOptions().AllCreateTableOption() {
			if tableOption.DefaultCollation() != nil {
				hasCollation = true
				break
			}
		}
	}
	if !hasCollation {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          advisor.NoCollation.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("Table %s does not have a collation specified", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(checker.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
