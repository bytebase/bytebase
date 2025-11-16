package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowRemoveTblCascadeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDisallowRemoveTblCascade, &StatementDisallowRemoveTblCascadeAdvisor{})
}

// StatementDisallowRemoveTblCascadeAdvisor is the advisor checking for disallow CASCADE option when removing tables.
type StatementDisallowRemoveTblCascadeAdvisor struct {
}

// Check checks for CASCADE option in DROP TABLE and TRUNCATE TABLE statements.
func (*StatementDisallowRemoveTblCascadeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowRemoveTblCascadeRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type statementDisallowRemoveTblCascadeRule struct {
	BaseRule
}

// Name returns the rule name.
func (*statementDisallowRemoveTblCascadeRule) Name() string {
	return "statement.disallow-remove-table-cascade"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementDisallowRemoveTblCascadeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Dropstmt":
		r.handleDropstmt(ctx.(*parser.DropstmtContext))
	case "Truncatestmt":
		r.handleTruncatestmt(ctx.(*parser.TruncatestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementDisallowRemoveTblCascadeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementDisallowRemoveTblCascadeRule) handleDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a DROP TABLE statement
	if ctx.Object_type_any_name() == nil || ctx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Check for CASCADE option
	if r.hasCascadeOption(ctx.Opt_drop_behavior()) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementDisallowCascade.Int32(),
			Title:   r.title,
			Content: "The use of CASCADE is not permitted when removing a table",
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

func (r *statementDisallowRemoveTblCascadeRule) handleTruncatestmt(ctx *parser.TruncatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check for CASCADE option
	if r.hasCascadeOption(ctx.Opt_drop_behavior()) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementDisallowCascade.Int32(),
			Title:   r.title,
			Content: "The use of CASCADE is not permitted when removing a table",
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// hasCascadeOption checks if the drop behavior is CASCADE
func (*statementDisallowRemoveTblCascadeRule) hasCascadeOption(ctx parser.IOpt_drop_behaviorContext) bool {
	if ctx == nil {
		return false
	}
	return ctx.CASCADE() != nil
}
