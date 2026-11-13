// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*WhereNoLeadingWildcardLikeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE, &WhereNoLeadingWildcardLikeAdvisor{})
}

// WhereNoLeadingWildcardLikeAdvisor is the advisor checking for no leading wildcard LIKE.
// The judgment lives in omni (oracle/v1review.StatementWhereNoLeadingWildcardLike); this adapter
// renders its findings as advice.
type WhereNoLeadingWildcardLikeAdvisor struct {
}

// Check checks for no leading wildcard LIKE.
func (*WhereNoLeadingWildcardLikeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oraclev1review.StatementWhereNoLeadingWildcardLike([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementLeadingWildcardLike.Int32(),
				Title:         title,
				Content:       "Avoid using leading wildcard LIKE.",
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
