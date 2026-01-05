package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &noSelectAllRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens: antlrAST.Tokens,
		}
		rule.SetBaseLine(stmtInfo.BaseLine())

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type noSelectAllRule struct {
	BaseRule
	tokens *antlr.CommonTokenStream
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
				stmtText := getTextFromTokens(r.tokens, ctx)
				if stmtText == "" {
					stmtText = "<unknown statement>"
				}
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    code.StatementSelectAll.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("\"%s\" uses SELECT all", stmtText),
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
