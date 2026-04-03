package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &NoLeadingWildcardLikeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &NoLeadingWildcardLikeAdvisor{})
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

	rule := &noLeadingWildcardLikeOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type noLeadingWildcardLikeOmniRule struct {
	OmniBaseRule
}

func (*noLeadingWildcardLikeOmniRule) Name() string {
	return "NoLeadingWildcardLikeRule"
}

func (r *noLeadingWildcardLikeOmniRule) OnStatement(node ast.Node) {
	text := strings.TrimSpace(r.StmtText)
	ast.Inspect(node, func(n ast.Node) bool {
		if like, ok := n.(*ast.LikeExpr); ok {
			r.checkLikeExpr(like, text)
		}
		return true
	})
}

func (r *noLeadingWildcardLikeOmniRule) checkLikeExpr(like *ast.LikeExpr, text string) {
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
				StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(like.Loc))),
			})
		}
	}
}
