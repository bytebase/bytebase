package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

func (*TableRequireCollationAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableRequireCollationChecker{
		level: level,
		title: string(ctx.Rule.Type),
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
			Status:  checker.level,
			Code:    advisor.NoCollation.Int32(),
			Title:   checker.title,
			Content: fmt.Sprintf("Table %s does not have a collation specified", tableName),
			StartPosition: &storepb.Position{
				Line: int32(checker.baseLine + ctx.GetStart().GetLine()),
			},
		})
	}
}
