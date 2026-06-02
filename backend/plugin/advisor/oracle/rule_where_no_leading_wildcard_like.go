// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"strings"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*WhereNoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &WhereNoLeadingWildcardLikeAdvisor{})
}

// WhereNoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type WhereNoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*WhereNoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewWhereNoLeadingWildcardLikeRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// WhereNoLeadingWildcardLikeRule is the rule implementation for no leading wildcard LIKE.
type WhereNoLeadingWildcardLikeRule struct {
	BaseRule

	currentDatabase string
}

// NewWhereNoLeadingWildcardLikeRule creates a new WhereNoLeadingWildcardLikeRule.
func NewWhereNoLeadingWildcardLikeRule(level storepb.Advice_Status, title string, currentDatabase string) *WhereNoLeadingWildcardLikeRule {
	return &WhereNoLeadingWildcardLikeRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*WhereNoLeadingWildcardLikeRule) Name() string {
	return "where.no-leading-wildcard-like"
}

// OnStatement checks LIKE predicates in the omni AST.
func (r *WhereNoLeadingWildcardLikeRule) OnStatement(node ast.Node) {
	omniWalk(node, func(n ast.Node) {
		like, ok := n.(*ast.LikeExpr)
		if !ok {
			return
		}
		pattern, ok := like.Pattern.(*ast.StringLiteral)
		if !ok || !strings.HasPrefix(pattern.Val, "%") {
			return
		}
		r.AddAdvice(
			r.level,
			code.StatementLeadingWildcardLike.Int32(),
			"Avoid using leading wildcard LIKE.",
			common.ConvertANTLRLineToPosition(r.locLine(like.Loc)),
		)
	})
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
