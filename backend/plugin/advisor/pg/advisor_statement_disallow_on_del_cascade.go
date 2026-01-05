package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementDisallowOnDelCascadeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE, &StatementDisallowOnDelCascadeAdvisor{})
}

// StatementDisallowOnDelCascadeAdvisor is the advisor checking for ON DELETE CASCADE.
type StatementDisallowOnDelCascadeAdvisor struct {
}

// Check checks for ON DELETE CASCADE.
func (*StatementDisallowOnDelCascadeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		rule := &statementDisallowOnDelCascadeRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens: antlrAST.Tokens,
		}

		checker := NewGenericChecker([]Rule{rule})
		rule.SetBaseLine(stmtInfo.BaseLine())
		checker.SetBaseLine(stmtInfo.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)

		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type statementDisallowOnDelCascadeRule struct {
	BaseRule
	tokens *antlr.CommonTokenStream
}

// Name returns the rule name.
func (*statementDisallowOnDelCascadeRule) Name() string {
	return "statement.disallow-on-delete-cascade"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementDisallowOnDelCascadeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Key_delete":
		r.handleKeyDelete(ctx.(*parser.Key_deleteContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementDisallowOnDelCascadeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementDisallowOnDelCascadeRule) handleKeyDelete(ctx *parser.Key_deleteContext) {
	// Check if this has CASCADE as the key_action
	if ctx.Key_action() != nil {
		keyAction := ctx.Key_action()
		// Check if this is CASCADE
		if keyAction.CASCADE() != nil {
			// Find the top-level statement context
			var stmtCtx antlr.ParserRuleContext
			current := ctx.GetParent()
			for current != nil {
				if isTopLevel(current) {
					if prc, ok := current.(antlr.ParserRuleContext); ok {
						stmtCtx = prc
					}
					break
				}
				current = current.GetParent()
			}

			if stmtCtx != nil {
				// Note: To match legacy pg-query behavior, we need to use GetLine() - 1
				// The legacy advisor uses pg_query's stmt_location which points to the position
				// right before the statement (often a newline), while ANTLR's GetLine() points
				// to the actual line where the statement starts. This creates an off-by-one
				// when there are statements before the CREATE TABLE.
				line := stmtCtx.GetStart().GetLine() - 1
				if line < 1 {
					line = 1
				}
				statementText := getTextFromTokens(r.tokens, stmtCtx)
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    code.StatementDisallowCascade.Int32(),
					Title:   r.title,
					Content: "The CASCADE option is not permitted for ON DELETE clauses",
					StartPosition: common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
						Line:   int32(line),
						Column: 0,
					}, statementText),
				})
			}
		}
	}
}
