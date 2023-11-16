package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*CharsetAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLCharsetAllowlist, &CharsetAllowlistAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLCharsetAllowlist, &CharsetAllowlistAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLCharsetAllowlist, &CharsetAllowlistAdvisor{})
}

// CharsetAllowlistAdvisor is the advisor checking for charset allowlist.
type CharsetAllowlistAdvisor struct {
}

// Check checks for charset allowlist.
func (*CharsetAllowlistAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &charsetAllowlistChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		allowList: make(map[string]bool),
	}
	for _, charset := range payload.List {
		checker.allowList[strings.ToLower(charset)] = true
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

type charsetAllowlistChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	allowList  map[string]bool
}

func (checker *charsetAllowlistChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (checker *charsetAllowlistChecker) EnterCreateDatabase(ctx *mysql.CreateDatabaseContext) {
	for _, option := range ctx.AllCreateDatabaseOption() {
		if option.DefaultCharset() != nil {
			charset := mysqlparser.NormalizeMySQLCharsetName(option.DefaultCharset().CharsetName())
			charset = strings.ToLower(charset)
			checker.checkCharset(charset, ctx.GetStart().GetLine())
			break
		}
	}
}

func (checker *charsetAllowlistChecker) checkCharset(charset string, lineNumber int) {
	if _, exists := checker.allowList[charset]; charset != "" && !exists {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.DisabledCharset,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" used disabled charset '%s'", checker.text, charset),
			Line:    checker.baseLine + lineNumber,
		})
	}
}

// EnterCreateTable is called when production createTable is entered.
func (checker *charsetAllowlistChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.CreateTableOptions() != nil {
		for _, option := range ctx.CreateTableOptions().AllCreateTableOption() {
			if option.DefaultCharset() != nil {
				charset := mysqlparser.NormalizeMySQLCharsetName(option.DefaultCharset().CharsetName())
				charset = strings.ToLower(charset)
				checker.checkCharset(charset, ctx.GetStart().GetLine())
				break
			}
		}
	}

	if ctx.TableElementList() != nil {
		for _, tableElement := range ctx.TableElementList().AllTableElement() {
			if tableElement.ColumnDefinition() != nil {
				if tableElement.ColumnDefinition() == nil {
					continue
				}
				columnDef := tableElement.ColumnDefinition()
				if columnDef.FieldDefinition() == nil || columnDef.FieldDefinition().DataType() == nil {
					continue
				}
				if columnDef.FieldDefinition().DataType().CharsetWithOptBinary() == nil {
					continue
				}
				charsetName := columnDef.FieldDefinition().DataType().CharsetWithOptBinary().CharsetName()
				charset := mysqlparser.NormalizeMySQLCharsetName(charsetName)
				charset = strings.ToLower(charset)
				checker.checkCharset(charset, ctx.GetStart().GetLine())
			}
		}
	}
}

// EnterAlterDatabase is called when production alterDatabase is entered.
func (checker *charsetAllowlistChecker) EnterAlterDatabase(ctx *mysql.AlterDatabaseContext) {
	for _, option := range ctx.AllAlterDatabaseOption() {
		if option.CreateDatabaseOption() == nil || option.CreateDatabaseOption().DefaultCharset() == nil {
			continue
		}
		charset := mysqlparser.NormalizeMySQLCharsetName(option.CreateDatabaseOption().DefaultCharset().CharsetName())
		charset = strings.ToLower(charset)
		checker.checkCharset(charset, ctx.GetStart().GetLine())
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *charsetAllowlistChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil || item.FieldDefinition() == nil {
			continue
		}
		if item.FieldDefinition().DataType() == nil {
			continue
		}
		if item.FieldDefinition().DataType().CharsetWithOptBinary() == nil {
			continue
		}
		charset := mysqlparser.NormalizeMySQLCharsetName(item.FieldDefinition().DataType().CharsetWithOptBinary().CharsetName())
		charset = strings.ToLower(charset)
		checker.checkCharset(charset, ctx.GetStart().GetLine())
	}
	// alter table option
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		if option == nil {
			continue
		}
		for _, tableOption := range option.AllCreateTableOption() {
			if tableOption == nil || tableOption.DefaultCharset() == nil {
				continue
			}
			charset := mysqlparser.NormalizeMySQLCharsetName(tableOption.DefaultCharset().CharsetName())
			charset = strings.ToLower(charset)
			checker.checkCharset(charset, ctx.GetStart().GetLine())
		}
	}
}
