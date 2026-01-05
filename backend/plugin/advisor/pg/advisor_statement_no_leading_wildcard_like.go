package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// Note: strings import is still needed for pattern checking (HasPrefix)

var (
	_ advisor.Advisor = (*StatementNoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &StatementNoLeadingWildcardLikeAdvisor{})
}

// StatementNoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type StatementNoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*StatementNoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		rule := &statementNoLeadingWildcardLikeRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens:             antlrAST.Tokens,
			reportedStatements: make(map[antlr.ParserRuleContext]bool),
		}
		rule.SetBaseLine(stmtInfo.BaseLine())

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type statementNoLeadingWildcardLikeRule struct {
	BaseRule
	tokens             *antlr.CommonTokenStream
	reportedStatements map[antlr.ParserRuleContext]bool
}

// Name returns the rule name.
func (*statementNoLeadingWildcardLikeRule) Name() string {
	return "statement.no-leading-wildcard-like"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementNoLeadingWildcardLikeRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "A_expr_like":
		r.handleAExprLike(ctx.(*parser.A_expr_likeContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementNoLeadingWildcardLikeRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementNoLeadingWildcardLikeRule) handleAExprLike(ctx *parser.A_expr_likeContext) {
	// Check if this is a LIKE or ILIKE expression (not SIMILAR TO)
	if ctx.LIKE() == nil && ctx.ILIKE() == nil {
		return
	}

	// Get the pattern (second A_expr_qual_op)
	qualOps := ctx.AllA_expr_qual_op()
	if len(qualOps) < 2 {
		return
	}

	pattern := qualOps[1]
	patternStr := extractPatternString(pattern)

	// Check if pattern starts with wildcard
	if patternStr != "" && (strings.HasPrefix(patternStr, "%") || strings.HasPrefix(patternStr, "_")) {
		// Find the top-level statement for reporting
		stmtCtx := findTopLevelStatement(ctx)
		if stmtCtx == nil {
			return
		}

		// Only report once per statement
		if r.reportedStatements[stmtCtx] {
			return
		}
		r.reportedStatements[stmtCtx] = true

		stmtText := getTextFromTokens(r.tokens, stmtCtx)
		if stmtText == "" {
			stmtText = "<unknown statement>"
		}
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementLeadingWildcardLike.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\" uses leading wildcard LIKE", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(stmtCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// extractPatternString extracts the string literal from a LIKE pattern expression
func extractPatternString(ctx parser.IA_expr_qual_opContext) string {
	if ctx == nil {
		return ""
	}

	// Walk down the expression tree to find the string constant
	// The structure is: A_expr_qual_op -> ... -> Aexprconst -> Sconst -> Anysconst -> Terminal
	var findSconst func(antlr.Tree) parser.ISconstContext
	findSconst = func(node antlr.Tree) parser.ISconstContext {
		if node == nil {
			return nil
		}

		// Check if this is a Sconst context
		if sconst, ok := node.(parser.ISconstContext); ok {
			return sconst
		}

		// Recursively check children
		if prCtx, ok := node.(antlr.ParserRuleContext); ok {
			for i := 0; i < prCtx.GetChildCount(); i++ {
				if result := findSconst(prCtx.GetChild(i)); result != nil {
					return result
				}
			}
		}

		return nil
	}

	sconstCtx := findSconst(ctx)
	if sconstCtx == nil {
		return ""
	}

	// Extract the string value from the Sconst context
	// The text includes quotes, so we need to remove them
	text := sconstCtx.GetText()
	if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
		return text[1 : len(text)-1]
	}

	return text
}

// findTopLevelStatement finds the top-level statement containing the given context
func findTopLevelStatement(ctx antlr.ParserRuleContext) antlr.ParserRuleContext {
	current := ctx
	for current != nil {
		parent := current.GetParent()
		if isTopLevel(parent) {
			return current
		}
		if prCtx, ok := parent.(antlr.ParserRuleContext); ok {
			current = prCtx
		} else {
			break
		}
	}
	return current
}
