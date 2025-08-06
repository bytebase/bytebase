package oracle

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
}

type StatementDisallowMixInDDLAdvisor struct {
}

func (*StatementDisallowMixInDDLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_SDL, storepb.PlanCheckRunConfig_DDL_GHOST:
	default:
		return nil, nil
	}
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewStatementDisallowMixInDDLRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList()
}

// StatementDisallowMixInDDLRule is the rule implementation for disallowing mix in DDL.
type StatementDisallowMixInDDLRule struct {
	BaseRule
}

// NewStatementDisallowMixInDDLRule creates a new StatementDisallowMixInDDLRule.
func NewStatementDisallowMixInDDLRule(level storepb.Advice_Status, title string) *StatementDisallowMixInDDLRule {
	return &StatementDisallowMixInDDLRule{
		BaseRule: NewBaseRule(level, title, 0),
	}
}

// Name returns the rule name.
func (*StatementDisallowMixInDDLRule) Name() string {
	return "statement.disallow-mix-in-ddl"
}

// OnEnter is called when the parser enters a rule context.
func (r *StatementDisallowMixInDDLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Unit_statement" {
		r.handleUnitStatement(ctx.(*parser.Unit_statementContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*StatementDisallowMixInDDLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementDisallowMixInDDLRule) handleUnitStatement(ctx *parser.Unit_statementContext) {
	if ctx.Data_manipulation_language_statements() != nil {
		r.AddAdvice(
			r.level,
			advisor.StatementDisallowMixDDLDML.Int32(),
			"Alter schema can only run DDL",
			common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		)
	}
}
