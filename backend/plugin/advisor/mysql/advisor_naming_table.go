package mysql

import (
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLNamingTableConvention, &NamingTableConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLNamingTableConvention, &NamingTableConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLNamingTableConvention, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	list, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingTableConventionChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		format:    format,
		maxLength: maxLength,
	}

	for _, stmt := range list {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}
	return checker.generateAdvice()
}

type namingTableConventionChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

func (checker *namingTableConventionChecker) generateAdvice() ([]advisor.Advice, error) {
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

// EnterCreateTable is called when production createTable is entered.
func (checker *namingTableConventionChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	checker.handleTableName(tableName, ctx.GetStart().GetLine())
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *namingTableConventionChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item.RENAME_SYMBOL() == nil {
			continue
		}
		if item.TableName() == nil {
			continue
		}
		_, tableName := mysqlparser.NormalizeMySQLTableName(item.TableName())
		checker.handleTableName(tableName, ctx.GetStart().GetLine())
	}
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (checker *namingTableConventionChecker) EnterRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	for _, pair := range ctx.AllRenamePair() {
		if pair.TableName() == nil {
			continue
		}
		_, tableName := mysqlparser.NormalizeMySQLTableName(pair.TableName())
		checker.handleTableName(tableName, ctx.GetStart().GetLine())
	}
}

func (checker *namingTableConventionChecker) handleTableName(tableName string, lineNumber int) {
	lineNumber += checker.baseLine
	if !checker.format.MatchString(tableName) {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.NamingTableConventionMismatch,
			Title:   checker.title,
			Content: fmt.Sprintf("`%s` mismatches table naming convention, naming format should be %q", tableName, checker.format),
			Line:    lineNumber,
		})
	}
	if checker.maxLength > 0 && len(tableName) > checker.maxLength {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.NamingTableConventionMismatch,
			Title:   checker.title,
			Content: fmt.Sprintf("`%s` mismatches table naming convention, its length should be within %d characters", tableName, checker.maxLength),
			Line:    lineNumber,
		})
	}
}
