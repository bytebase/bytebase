package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexPrimaryKeyTypeAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, &IndexPrimaryKeyTypeAllowlistAdvisor{})
}

// IndexPrimaryKeyTypeAllowlistAdvisor is the advisor checking for primary key
// type allowlist. The judgment lives in omni
// (pg/v1review.IndexPrimaryKeyTypeAllowlist); this adapter passes the
// payload's list and renders the findings as advice.
type IndexPrimaryKeyTypeAllowlistAdvisor struct {
}

// Check checks for primary key type allowlist.
func (*IndexPrimaryKeyTypeAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// A rule without a string array payload panics here, and advisor.Check
	// reports the panic as the rule's error.
	allowlist := checkCtx.Rule.GetStringArrayPayload().List

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for i, node := range nodes {
		for _, f := range pgv1review.IndexPrimaryKeyTypeAllowlist([]ast.Node{node}, allowlist) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.IndexPKType.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The column %q is one of the primary key, but its type %q is not in allowlist", f.Name, f.Value),
				StartPosition: findingPosition(stmts[i], f),
			})
		}
	}
	return adviceList, nil
}
