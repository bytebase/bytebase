package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDisallowOnDelCascadeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementDisallowOnDelCascade, &StatementDisallowOnDelCascadeAdvisor{})
}

// StatementDisallowOnDelCascadeAdvisor is the advisor checking for ON DELETE CASCADE.
type StatementDisallowOnDelCascadeAdvisor struct {
}

// Check checks for ON DELETE CASCADE.
func (*StatementDisallowOnDelCascadeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementDisallowOnDelCascadeRule{
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

type statementDisallowOnDelCascadeRule struct {
	BaseRule
	statementsText string
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
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    code.StatementDisallowCascade.Int32(),
					Title:   r.title,
					Content: "The CASCADE option is not permitted for ON DELETE clauses",
					StartPosition: common.ConvertANTLRPositionToPosition(&common.ANTLRPosition{
						Line:   int32(line),
						Column: 0,
					}, r.statementsText),
				})
			}
		}
	}
}
