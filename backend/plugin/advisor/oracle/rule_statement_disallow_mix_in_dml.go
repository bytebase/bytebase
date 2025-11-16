package oracle

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
}

type StatementDisallowMixInDMLAdvisor struct {
}

func (*StatementDisallowMixInDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DML:
	default:
		return nil, nil
	}

	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewStatementDisallowMixInDMLRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// StatementDisallowMixInDMLRule is the rule implementation for disallowing mix in DML.
type StatementDisallowMixInDMLRule struct {
	BaseRule
}

// NewStatementDisallowMixInDMLRule creates a new StatementDisallowMixInDMLRule.
func NewStatementDisallowMixInDMLRule(level storepb.Advice_Status, title string) *StatementDisallowMixInDMLRule {
	return &StatementDisallowMixInDMLRule{
		BaseRule: NewBaseRule(level, title, 0),
	}
}

// Name returns the rule name.
func (*StatementDisallowMixInDMLRule) Name() string {
	return "statement.disallow-mix-in-dml"
}

// OnEnter is called when the parser enters a rule context.
func (r *StatementDisallowMixInDMLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Unit_statement" {
		r.handleUnitStatement(ctx.(*parser.Unit_statementContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*StatementDisallowMixInDMLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementDisallowMixInDMLRule) handleUnitStatement(ctx *parser.Unit_statementContext) {
	if ctx.Data_manipulation_language_statements() == nil {
		r.AddAdvice(
			r.level,
			code.StatementDisallowMixDDLDML.Int32(),
			"Data change can only run DML",
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}
