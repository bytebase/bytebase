package mssql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/tsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*StatementDisallowMixInDMLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleStatementDisallowMixInDML, &StatementDisallowMixInDMLAdvisor{})
}

type StatementDisallowMixInDMLAdvisor struct {
}

func (*StatementDisallowMixInDMLAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	switch checkCtx.ChangeType {
	case storepb.PlanCheckRunConfig_DML:
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
	title := string(checkCtx.Rule.Type)

	// Create the rule
	rule := NewStatementDisallowMixInDMLRule(level, title, checkCtx.ChangeType)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// StatementDisallowMixInDMLRule is the rule checking for disallow mix in DML.
type StatementDisallowMixInDMLRule struct {
	BaseRule
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType
}

// NewStatementDisallowMixInDMLRule creates a new StatementDisallowMixInDMLRule.
func NewStatementDisallowMixInDMLRule(level storepb.Advice_Status, title string, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) *StatementDisallowMixInDMLRule {
	return &StatementDisallowMixInDMLRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		changeType: changeType,
	}
}

// Name returns the rule name.
func (*StatementDisallowMixInDMLRule) Name() string {
	return "StatementDisallowMixInDMLRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementDisallowMixInDMLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Sql_clauses" {
		r.enterSQLClauses(ctx.(*parser.Sql_clausesContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementDisallowMixInDMLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *StatementDisallowMixInDMLRule) enterSQLClauses(ctx *parser.Sql_clausesContext) {
	if !tsqlparser.IsTopLevel(ctx.GetParent()) {
		return
	}
	var isDML bool
	if ctx.Dml_clause() != nil {
		isDML = true
	}

	if !isDML {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       "Data change can only run DML",
			Code:          code.StatementDisallowMixDDLDML.Int32(),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
