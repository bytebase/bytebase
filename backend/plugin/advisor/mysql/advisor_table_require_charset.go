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
	_ advisor.Advisor = (*TableRequireCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableRequireCharset, &TableRequireCharsetAdvisor{})
}

// TableRequireCharsetAdvisor is the advisor checking for require charset.
type TableRequireCharsetAdvisor struct {
}

func (*TableRequireCharsetAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableRequireCharsetChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.adviceList, nil
}

type tableRequireCharsetChecker struct {
	*mysql.BaseMySQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	baseLine   int
}

func (checker *tableRequireCharsetChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if tableName == "" {
		return
	}

	hasCharset := false
	if ctx.CreateTableOptions() != nil {
		for _, tableOption := range ctx.CreateTableOptions().AllCreateTableOption() {
			if tableOption.DefaultCharset() != nil {
				hasCharset = true
				break
			}
		}
	}
	if !hasCharset {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  checker.level,
			Code:    advisor.NoCharset.Int32(),
			Title:   checker.title,
			Content: fmt.Sprintf("Table %s does not have a character set specified", tableName),
			StartPosition: &storepb.Position{
				Line: int32(checker.baseLine + ctx.GetStart().GetLine()),
			},
		})
	}
}
