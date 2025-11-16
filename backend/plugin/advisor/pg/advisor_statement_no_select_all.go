package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementNoSelectAll, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &noSelectAllRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		statementsText: checkCtx.Statements,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type noSelectAllRule struct {
	BaseRule
	statementsText string
}

// Name returns the rule name.
func (*noSelectAllRule) Name() string {
	return "statement.no-select-all"
}

// OnEnter is called when the parser enters a rule context.
func (r *noSelectAllRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Simple_select_pramary":
		r.handleSimpleSelectPramary(ctx.(*parser.Simple_select_pramaryContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*noSelectAllRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *noSelectAllRule) handleSimpleSelectPramary(ctx *parser.Simple_select_pramaryContext) {
	// Check if this is a SELECT statement with target list
	if ctx.SELECT() == nil {
		return
	}

	// Check the target list for * (asterisk)
	if ctx.Opt_target_list() != nil && ctx.Opt_target_list().Target_list() != nil {
		targetList := ctx.Opt_target_list().Target_list()
		allTargets := targetList.AllTarget_el()

		for _, target := range allTargets {
			// Check if target is a Target_star (SELECT *)
			if _, ok := target.(*parser.Target_starContext); ok {
				// Find the top-level statement context to get the full statement text
				var stmtCtx antlr.ParserRuleContext
				parent := ctx.GetParent()
				for parent != nil {
					// Look for top-level statement contexts
					switch p := parent.(type) {
					case *parser.SelectstmtContext:
						if isTopLevel(p.GetParent()) {
							stmtCtx = p
						}
					case *parser.InsertstmtContext:
						if isTopLevel(p.GetParent()) {
							stmtCtx = p
						}
					}
					if stmtCtx != nil {
						break
					}
					parent = parent.GetParent()
				}

				// If we found a top-level statement, extract its text
				var stmtText string
				var line int
				if stmtCtx != nil {
					stmtText = extractStatementText(r.statementsText, stmtCtx.GetStart().GetLine(), stmtCtx.GetStop().GetLine())
					line = stmtCtx.GetStart().GetLine()
				} else {
					// Fallback to the simple_select context
					stmtText = extractStatementText(r.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
					line = ctx.GetStart().GetLine()
				}

				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    code.StatementSelectAll.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("\"%s\" uses SELECT all", stmtText),
					StartPosition: &storepb.Position{
						Line:   int32(line),
						Column: 0,
					},
				})
				return
			}
		}
	}
}
