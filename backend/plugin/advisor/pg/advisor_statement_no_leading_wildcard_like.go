package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

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

	rule := &statementNoLeadingWildcardLikeRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementNoLeadingWildcardLikeRule struct {
	OmniBaseRule
}

func (*statementNoLeadingWildcardLikeRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE)
}

func (r *statementNoLeadingWildcardLikeRule) OnStatement(node ast.Node) {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		expr, ok := n.(*ast.A_Expr)
		if !ok {
			return true
		}
		if expr.Kind != ast.AEXPR_LIKE && expr.Kind != ast.AEXPR_ILIKE {
			return true
		}
		// Check if right operand is a string constant starting with '%' or '_'.
		if ac, ok := expr.Rexpr.(*ast.A_Const); ok {
			if sv, ok := ac.Val.(*ast.String); ok {
				if strings.HasPrefix(sv.Str, "%") || strings.HasPrefix(sv.Str, "_") {
					found = true
					r.AddAdvice(&storepb.Advice{
						Status:  r.Level,
						Code:    code.StatementLeadingWildcardLike.Int32(),
						Title:   r.Title,
						Content: fmt.Sprintf("\"%s\" uses leading wildcard LIKE", r.TrimmedStmtText()),
						StartPosition: &storepb.Position{
							Line:   r.ContentStartLine(),
							Column: 0,
						},
					})
					return false
				}
			}
		}
		return true
	})
}
