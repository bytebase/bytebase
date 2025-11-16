package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleCollationAllowlist, &CollationAllowlistAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleCollationAllowlist, &CollationAllowlistAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleCollationAllowlist, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor is the advisor checking for collation allowlist.
type CollationAllowlistAdvisor struct {
}

// Check checks for collation allowlist.
func (*CollationAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewCollationAllowlistRule(level, string(checkCtx.Rule.Type), payload.List)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		// Set text will be handled in the Query node
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// CollationAllowlistRule checks for collation allowlist.
type CollationAllowlistRule struct {
	BaseRule
	allowList map[string]bool
	text      string
}

// NewCollationAllowlistRule creates a new CollationAllowlistRule.
func NewCollationAllowlistRule(level storepb.Advice_Status, title string, allowList []string) *CollationAllowlistRule {
	rule := &CollationAllowlistRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		allowList: make(map[string]bool),
	}
	for _, collation := range allowList {
		rule.allowList[strings.ToLower(collation)] = true
	}
	return rule
}

// Name returns the rule name.
func (*CollationAllowlistRule) Name() string {
	return "CollationAllowlistRule"
}

// SetText sets the query text for error reporting.
func (r *CollationAllowlistRule) SetText(text string) {
	r.text = text
}

// OnEnter is called when entering a parse tree node.
func (r *CollationAllowlistRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		r.checkQuery(ctx.(*mysql.QueryContext))
	case NodeTypeCreateDatabase:
		r.checkCreateDatabase(ctx.(*mysql.CreateDatabaseContext))
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterDatabase:
		r.checkAlterDatabase(ctx.(*mysql.AlterDatabaseContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*CollationAllowlistRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *CollationAllowlistRule) checkQuery(ctx *mysql.QueryContext) {
	r.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (r *CollationAllowlistRule) checkCreateDatabase(ctx *mysql.CreateDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	for _, option := range ctx.AllCreateDatabaseOption() {
		if option != nil && option.DefaultCollation() != nil && option.DefaultCollation().CollationName() != nil {
			collation := mysqlparser.NormalizeMySQLCollationName(option.DefaultCollation().CollationName())
			r.checkCollation(collation, ctx.GetStart().GetLine())
		}
	}
}

func (r *CollationAllowlistRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateTableOptions() != nil {
		for _, option := range ctx.CreateTableOptions().AllCreateTableOption() {
			if option != nil && option.DefaultCollation() != nil && option.DefaultCollation().CollationName() != nil {
				collation := mysqlparser.NormalizeMySQLCollationName(option.DefaultCollation().CollationName())
				r.checkCollation(collation, option.GetStart().GetLine())
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
					r.checkCollation(collation, columnDef.GetStart().GetLine())
				}
			}
		}
	}
}

func (r *CollationAllowlistRule) checkAlterDatabase(ctx *mysql.AlterDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	for _, option := range ctx.AllAlterDatabaseOption() {
		if option == nil || option.CreateDatabaseOption() == nil || option.CreateDatabaseOption().DefaultCollation() == nil || option.CreateDatabaseOption().DefaultCollation().CollationName() == nil {
			continue
		}
		charset := mysqlparser.NormalizeMySQLCollationName(option.CreateDatabaseOption().DefaultCollation().CollationName())
		r.checkCollation(charset, ctx.GetStart().GetLine())
	}
}

func (r *CollationAllowlistRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
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
			r.checkCollation(collation, item.GetStart().GetLine())
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
			r.checkCollation(collation, tableOption.GetStart().GetLine())
		}
	}
}

func (r *CollationAllowlistRule) checkCollation(collation string, lineNumber int) {
	collation = strings.ToLower(collation)
	if _, exists := r.allowList[collation]; collation != "" && !exists {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.DisabledCollation.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" used disabled collation '%s'", r.text, collation),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + lineNumber),
		})
	}
}
