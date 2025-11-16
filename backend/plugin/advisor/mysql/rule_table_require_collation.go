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
	_ advisor.Advisor = (*TableRequireCollationAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableRequireCollation, &TableRequireCollationAdvisor{})
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

	// Create the rule
	rule := NewTableRequireCollationRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableRequireCollationRule checks that tables have collation specified.
type TableRequireCollationRule struct {
	BaseRule
}

// NewTableRequireCollationRule creates a new TableRequireCollationRule.
func NewTableRequireCollationRule(level storepb.Advice_Status, title string) *TableRequireCollationRule {
	return &TableRequireCollationRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*TableRequireCollationRule) Name() string {
	return "TableRequireCollationRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableRequireCollationRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateTable {
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableRequireCollationRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableRequireCollationRule) checkCreateTable(ctx *mysql.CreateTableContext) {
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
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NoCollation.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Table %s does not have a collation specified", tableName),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
