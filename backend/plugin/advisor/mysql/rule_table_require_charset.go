package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableRequireCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableRequireCharset, &TableRequireCharsetAdvisor{})
}

// TableRequireCharsetAdvisor is the advisor checking for require charset.
type TableRequireCharsetAdvisor struct {
}

func (*TableRequireCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableRequireCharsetRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableRequireCharsetRule checks that tables have charset specified.
type TableRequireCharsetRule struct {
	BaseRule
}

// NewTableRequireCharsetRule creates a new TableRequireCharsetRule.
func NewTableRequireCharsetRule(level storepb.Advice_Status, title string) *TableRequireCharsetRule {
	return &TableRequireCharsetRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*TableRequireCharsetRule) Name() string {
	return "TableRequireCharsetRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableRequireCharsetRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateTable {
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableRequireCharsetRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableRequireCharsetRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NoCharset.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %s does not have a character set specified", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
