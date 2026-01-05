package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementDisallowAddColumnWithDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT, &StatementDisallowAddColumnWithDefaultAdvisor{})
}

// StatementDisallowAddColumnWithDefaultAdvisor is the advisor checking for to disallow add column with default.
type StatementDisallowAddColumnWithDefaultAdvisor struct {
}

// Check checks for to disallow add column with default.
func (*StatementDisallowAddColumnWithDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowAddColumnWithDefaultRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

type statementDisallowAddColumnWithDefaultRule struct {
	BaseRule
}

// Name returns the rule name.
func (*statementDisallowAddColumnWithDefaultRule) Name() string {
	return "statement.disallow-add-column-with-default"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementDisallowAddColumnWithDefaultRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementDisallowAddColumnWithDefaultRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementDisallowAddColumnWithDefaultRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check all alter table commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// Check for ADD COLUMN
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				columnDef := cmd.ColumnDef()
				// Check if the column has a DEFAULT constraint
				if columnDef.Colquallist() != nil {
					allConstraints := columnDef.Colquallist().AllColconstraint()
					for _, constraint := range allConstraints {
						if constraint.Colconstraintelem() != nil {
							constraintElem := constraint.Colconstraintelem()
							// Check for DEFAULT constraint
							if constraintElem.DEFAULT() != nil {
								r.AddAdvice(&storepb.Advice{
									Status:  r.level,
									Code:    code.StatementAddColumnWithDefault.Int32(),
									Title:   r.title,
									Content: "Adding column with DEFAULT will locked the whole table and rewriting each rows",
									StartPosition: &storepb.Position{
										Line:   int32(ctx.GetStart().GetLine()),
										Column: 0,
									},
								})
								return
							}
						}
					}
				}
			}
		}
	}
}
