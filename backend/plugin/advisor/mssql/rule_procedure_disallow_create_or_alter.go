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
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleProcedureDisallowCreate, &ProcedureDisallowCreateOrAlterAdvisor{})
}

type ProcedureDisallowCreateOrAlterAdvisor struct{}

func (*ProcedureDisallowCreateOrAlterAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewProcedureDisallowCreateOrAlterRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// ProcedureDisallowCreateOrAlterRule is the rule for disallowing procedure creation or alteration.
type ProcedureDisallowCreateOrAlterRule struct {
	BaseRule
}

// NewProcedureDisallowCreateOrAlterRule creates a new ProcedureDisallowCreateOrAlterRule.
func NewProcedureDisallowCreateOrAlterRule(level storepb.Advice_Status, title string) *ProcedureDisallowCreateOrAlterRule {
	return &ProcedureDisallowCreateOrAlterRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ProcedureDisallowCreateOrAlterRule) Name() string {
	return "ProcedureDisallowCreateOrAlterRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ProcedureDisallowCreateOrAlterRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeCreateOrAlterProcedure {
		r.enterCreateOrAlterProcedure(ctx.(*parser.Create_or_alter_procedureContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ProcedureDisallowCreateOrAlterRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *ProcedureDisallowCreateOrAlterRule) enterCreateOrAlterProcedure(ctx *parser.Create_or_alter_procedureContext) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.DisallowCreateProcedure.Int32(),
		Title:         r.title,
		Content:       "Creating or altering procedures is prohibited",
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}
