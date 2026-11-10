package oceanbase

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oceanbase/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &NoLeadingWildcardLikeAdvisor{})
}

// NoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
type NoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*NoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &noLeadingWildcardLikeRule{
		BaseRule: BaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunRules(checkCtx.ParsedStatements, []Rule{rule}), nil
}

type noLeadingWildcardLikeRule struct {
	BaseRule
}

func (*noLeadingWildcardLikeRule) Name() string {
	return "NoLeadingWildcardLikeRule"
}

func (r *noLeadingWildcardLikeRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	ast.Inspect(node, func(n ast.Node) bool {
		if like, ok := n.(*ast.LikeExpr); ok {
			r.checkLikeExpr(like, text)
		}
		return true
	})
}

func (r *noLeadingWildcardLikeRule) checkLikeExpr(like *ast.LikeExpr, text string) {
	if like.Pattern == nil {
		return
	}
	if str, ok := like.Pattern.(*ast.StringLit); ok {
		if strings.HasPrefix(str.Value, "%") {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.StatementLeadingWildcardLike.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("\"%s\" uses leading wildcard LIKE", text),
				StartPosition: base.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(like.Loc))),
			})
		}
	}
}
