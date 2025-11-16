// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleTableNameNoKeyword, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewNamingTableNoKeywordRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// NamingTableNoKeywordRule checks for table naming without keywords.
type NamingTableNoKeywordRule struct {
	BaseRule
}

// NewNamingTableNoKeywordRule creates a new NamingTableNoKeywordRule.
func NewNamingTableNoKeywordRule(level storepb.Advice_Status, title string) *NamingTableNoKeywordRule {
	return &NamingTableNoKeywordRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NamingTableNoKeywordRule) Name() string {
	return "NamingTableNoKeywordRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingTableNoKeywordRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingTableNoKeywordRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *NamingTableNoKeywordRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Object_name().GetO())
	if snowsqlparser.IsSnowflakeKeyword(tableName, false) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NameIsKeywordIdentifier.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
		})
	}
}

func (r *NamingTableNoKeywordRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	if ctx.RENAME() == nil {
		return
	}

	tableName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Object_name(1).GetO())
	if snowsqlparser.IsSnowflakeKeyword(tableName, false) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NameIsKeywordIdentifier.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
		})
	}
}
