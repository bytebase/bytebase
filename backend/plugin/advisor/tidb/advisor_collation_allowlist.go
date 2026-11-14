package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor checks for collation allowlist. The judgment
// lives in omni (tidb/v1review.SystemCollationAllowlist); this adapter passes
// the rule's payload and renders the findings as advice.
type CollationAllowlistAdvisor struct {
}

// Check checks for collation allowlist.
func (*CollationAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	allowList := stringArrayPayload.List

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.SystemCollationAllowlist([]ast.Node{stmt.Node}, allowList) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.DisabledCollation.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" used disabled collation '%s'", stmt.TrimmedText(), f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(allowlistFindingLine(stmt, f)),
			})
		}
	}
	return adviceList, nil
}
