package mysql

import (
	"context"
	"fmt"
	"regexp"

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
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking the MySQLTableDropNamingConvention rule.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for drop table naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	list, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableDropNamingConventionRule(level, string(checkCtx.Rule.Type), format)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range list {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableDropNamingConventionRule checks for drop table naming convention.
type TableDropNamingConventionRule struct {
	BaseRule
	format *regexp.Regexp
}

// NewTableDropNamingConventionRule creates a new TableDropNamingConventionRule.
func NewTableDropNamingConventionRule(level storepb.Advice_Status, title string, format *regexp.Regexp) *TableDropNamingConventionRule {
	return &TableDropNamingConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		format: format,
	}
}

// Name returns the rule name.
func (*TableDropNamingConventionRule) Name() string {
	return "TableDropNamingConventionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableDropNamingConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeDropTable {
		r.checkDropTable(ctx.(*mysql.DropTableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableDropNamingConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableDropNamingConventionRule) checkDropTable(ctx *mysql.DropTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		if !r.format.MatchString(tableName) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableDropNamingConventionMismatch.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tableName, r.format),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}
