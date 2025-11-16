package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleIdentifierNoKeyword, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewNamingIdentifierNoKeywordRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// NamingIdentifierNoKeywordRule is the rule for identifier naming convention without keyword.
type NamingIdentifierNoKeywordRule struct {
	BaseRule
}

// NewNamingIdentifierNoKeywordRule creates a new NamingIdentifierNoKeywordRule.
func NewNamingIdentifierNoKeywordRule(level storepb.Advice_Status, title string) *NamingIdentifierNoKeywordRule {
	return &NamingIdentifierNoKeywordRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NamingIdentifierNoKeywordRule) Name() string {
	return "NamingIdentifierNoKeywordRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingIdentifierNoKeywordRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeID {
		r.enterID(ctx.(*parser.Id_Context))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingIdentifierNoKeywordRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *NamingIdentifierNoKeywordRule) enterID(ctx *parser.Id_Context) {
	if ctx == nil {
		return
	}

	parent := ctx.GetParent()
	switch parent.(type) {
	case *parser.Column_definitionContext:
	case *parser.Table_constraintContext:
	case *parser.Create_schemaContext:
	case *parser.Create_databaseContext:
	case *parser.Create_indexContext:
	case *parser.Table_nameContext:
	default:
		return
	}
	if ctx.GetText() == "" {
		return
	}

	_, normalizedID := tsqlparser.NormalizeTSQLIdentifier(ctx)
	if tsqlparser.IsTSQLReservedKeyword(normalizedID, false) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NameIsKeywordIdentifier.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Identifier [%s] is a keyword identifier and should be avoided.", normalizedID),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
