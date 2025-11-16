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
	_ advisor.Advisor = (*StatementDisallowMixInDDLAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleStatementDisallowMixInDDL, &StatementDisallowMixInDDLAdvisor{})
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
	title := string(checkCtx.Rule.Type)

	// Create the rule
	rule := NewStatementDisallowMixInDDLRule(level, title, checkCtx.ChangeType)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// StatementDisallowMixInDDLRule is the rule checking for disallow mix in DDL.
type StatementDisallowMixInDDLRule struct {
	BaseRule
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType
}

// NewStatementDisallowMixInDDLRule creates a new StatementDisallowMixInDDLRule.
func NewStatementDisallowMixInDDLRule(level storepb.Advice_Status, title string, changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) *StatementDisallowMixInDDLRule {
	return &StatementDisallowMixInDDLRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		changeType: changeType,
	}
}

// Name returns the rule name.
func (*StatementDisallowMixInDDLRule) Name() string {
	return "StatementDisallowMixInDDLRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementDisallowMixInDDLRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Sql_clauses" {
		r.enterSQLClauses(ctx.(*parser.Sql_clausesContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementDisallowMixInDDLRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *StatementDisallowMixInDDLRule) enterSQLClauses(ctx *parser.Sql_clausesContext) {
	if !tsqlparser.IsTopLevel(ctx.GetParent()) {
		return
	}
	var isDML bool
	if ctx.Dml_clause() != nil {
		isDML = true
	}

	if isDML {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Title:         r.title,
			Content:       "Alter schema can only run DDL",
			Code:          code.StatementDisallowMixDDLDML.Int32(),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
