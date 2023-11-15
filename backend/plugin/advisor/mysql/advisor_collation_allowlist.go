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
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	// only for mysqlwip test.
	advisor.Register(storepb.Engine_ENGINE_UNSPECIFIED, advisor.MySQLCollationAllowlist, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor is the advisor checking for collation allowlist.
type CollationAllowlistAdvisor struct {
}

// Check checks for collation allowlist.
func (*CollationAllowlistAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
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
	checker := &collationAllowlistChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		allowList: make(map[string]bool),
	}
	for _, collation := range payload.List {
		checker.allowList[strings.ToLower(collation)] = true
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

type collationAllowlistChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine   int
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	allowList  map[string]bool
}

func (checker *collationAllowlistChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (checker *collationAllowlistChecker) EnterCreateDatabase(ctx *mysql.CreateDatabaseContext) {
	for _, option := range ctx.AllCreateDatabaseOption() {
		if option != nil && option.DefaultCollation() != nil && option.DefaultCollation().CollationName() != nil {
			collation := mysqlparser.NormalizeMySQLCollationName(option.DefaultCollation().CollationName())
			checker.checkCollation(collation, ctx.GetStart().GetLine())
		}
	}
}

func (checker *collationAllowlistChecker) checkCollation(collation string, lineNumber int) {
	collation = strings.ToLower(collation)
	if _, exists := checker.allowList[collation]; collation != "" && !exists {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.DisabledCollation,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" used disabled collation '%s'", checker.text, collation),
			Line:    checker.baseLine + lineNumber,
		})
	}
}

// EnterCreateTable is called when production createTable is entered.
func (checker *collationAllowlistChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.CreateTableOptions() != nil {
		for _, option := range ctx.CreateTableOptions().AllCreateTableOption() {
			if option != nil && option.DefaultCollation() != nil && option.DefaultCollation().CollationName() != nil {
				collation := mysqlparser.NormalizeMySQLCollationName(option.DefaultCollation().CollationName())
				checker.checkCollation(collation, option.GetStart().GetLine())
			}
		}
	}

	if ctx.TableElementList() != nil {
		for _, tableElement := range ctx.TableElementList().AllTableElement() {
			if tableElement == nil || tableElement.ColumnDefinition() == nil {
				continue
			}
			columnDef := tableElement.ColumnDefinition()
			if columnDef.FieldDefinition() == nil {
				continue
			}
			if columnDef.FieldDefinition().AllColumnAttribute() == nil {
				continue
			}
			for _, attr := range columnDef.FieldDefinition().AllColumnAttribute() {
				if attr != nil && attr.Collate() != nil && attr.Collate().CollationName() != nil {
					collation := mysqlparser.NormalizeMySQLCollationName(attr.Collate().CollationName())
					checker.checkCollation(collation, tableElement.GetStart().GetLine())
				}
			}
		}
	}
}

// EnterAlterDatabase is called when production alterDatabase is entered.
func (checker *collationAllowlistChecker) EnterAlterDatabase(ctx *mysql.AlterDatabaseContext) {
	for _, option := range ctx.AllAlterDatabaseOption() {
		if option == nil || option.CreateDatabaseOption() == nil || option.CreateDatabaseOption().DefaultCollation() == nil || option.CreateDatabaseOption().DefaultCollation().CollationName() == nil {
			continue
		}
		charset := mysqlparser.NormalizeMySQLCollationName(option.CreateDatabaseOption().DefaultCollation().CollationName())
		checker.checkCollation(charset, ctx.GetStart().GetLine())
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *collationAllowlistChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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
		if item == nil || item.FieldDefinition() == nil {
			continue
		}
		for _, attr := range item.FieldDefinition().AllColumnAttribute() {
			if attr == nil || attr.Collate() == nil || attr.Collate().CollationName() == nil {
				continue
			}
			collation := mysqlparser.NormalizeMySQLCollationName(attr.Collate().CollationName())
			checker.checkCollation(collation, item.GetStart().GetLine())
		}
	}
	// alter table option
	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		if option == nil {
			continue
		}
		for _, tableOption := range option.AllCreateTableOption() {
			if tableOption == nil {
				continue
			}
			if tableOption.DefaultCollation() == nil || tableOption.DefaultCollation().CollationName() == nil {
				continue
			}
			collation := mysqlparser.NormalizeMySQLCollationName(tableOption.DefaultCollation().CollationName())
			checker.checkCollation(collation, tableOption.GetStart().GetLine())
		}
	}
}
