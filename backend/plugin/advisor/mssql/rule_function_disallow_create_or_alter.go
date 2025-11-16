package mssql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleFunctionDisallowCreate, &FunctionDisallowCreateOrAlterAdvisor{})
}

type FunctionDisallowCreateOrAlterAdvisor struct{}

// Check implements advisor.Advisor.
func (*FunctionDisallowCreateOrAlterAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewFunctionDisallowCreateOrAlterRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// FunctionDisallowCreateOrAlterRule is the rule for disallowing function creation or alteration.
type FunctionDisallowCreateOrAlterRule struct {
	BaseRule
}

// NewFunctionDisallowCreateOrAlterRule creates a new FunctionDisallowCreateOrAlterRule.
func NewFunctionDisallowCreateOrAlterRule(level storepb.Advice_Status, title string) *FunctionDisallowCreateOrAlterRule {
	return &FunctionDisallowCreateOrAlterRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*FunctionDisallowCreateOrAlterRule) Name() string {
	return "FunctionDisallowCreateOrAlterRule"
}

// OnEnter is called when entering a parse tree node.
func (r *FunctionDisallowCreateOrAlterRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateOrAlterFunction {
		r.enterCreateOrAlterFunction(ctx.(*parser.Create_or_alter_functionContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*FunctionDisallowCreateOrAlterRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *FunctionDisallowCreateOrAlterRule) enterCreateOrAlterFunction(ctx *parser.Create_or_alter_functionContext) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.DisallowCreateFunction.Int32(),
		Title:         r.title,
		Content:       "Creating or altering functions is prohibited",
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}
