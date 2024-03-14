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
	_ advisor.Advisor = (*TableDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableDisallowSetCharset, &TableDisallowSetCharsetAdvisor{})
}

type TableDisallowSetCharsetAdvisor struct {
}

func (*TableDisallowSetCharsetAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowSetCharsetChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

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

type tableDisallowSetCharsetChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
}

func (checker *tableDisallowSetCharsetChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *tableDisallowSetCharsetChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.CreateTableOptions() != nil {
		for _, option := range ctx.CreateTableOptions().AllCreateTableOption() {
			if option.DefaultCharset() != nil {
				checker.adviceList = append(checker.adviceList, advisor.Advice{
					Status:  checker.level,
					Code:    advisor.DisallowSetCharset,
					Title:   checker.title,
					Content: fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", checker.text),
					Line:    checker.baseLine + ctx.GetStart().GetLine(),
				})
			}
		}
	}
}

func (checker *tableDisallowSetCharsetChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}

	alterList := ctx.AlterTableActions().AlterCommandList().AlterList()
	if alterList == nil {
		return
	}
	for _, alterListItem := range alterList.AllAlterListItem() {
		if alterListItem == nil {
			continue
		}

		if alterListItem.Charset() != nil {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.DisallowSetCharset,
				Title:   checker.title,
				Content: fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", checker.text),
				Line:    checker.baseLine + ctx.GetStart().GetLine(),
			})
		}
	}
}
