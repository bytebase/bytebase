// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking for table drop with naming convention.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for table drop with naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := NewTableDropNamingConventionRule(level, string(checkCtx.Rule.Type), format)
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// TableDropNamingConventionRule checks for table drop naming convention.
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
		r.enterDropTable(ctx.(*parser.Drop_tableContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableDropNamingConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *TableDropNamingConventionRule) enterDropTable(ctx *parser.Drop_tableContext) {
	normalizedObjectName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Object_name().GetO())
	if !r.format.MatchString(normalizedObjectName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          advisor.TableDropNamingConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("%q mismatches drop table naming convention, naming format should be %q", normalizedObjectName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.Object_name().GetO().GetStart().GetLine()),
		})
	}
}
