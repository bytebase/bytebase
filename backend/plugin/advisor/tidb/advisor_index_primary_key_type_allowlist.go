package tidb

import (
	"context"
	"fmt"

	tidbv1review "github.com/bytebase/omni/tidb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*IndexPrimaryKeyTypeAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, &IndexPrimaryKeyTypeAllowlistAdvisor{})
}

// IndexPrimaryKeyTypeAllowlistAdvisor is the advisor checking for primary
// key type allowlist. The judgment lives in omni
// (tidb/v1review.IndexPrimaryKeyTypeAllowlist); this adapter passes the
// allowlist, the script and the pre-change schema and renders the findings
// as advice.
type IndexPrimaryKeyTypeAllowlistAdvisor struct {
}

// Check checks for primary key type allowlist.
func (*IndexPrimaryKeyTypeAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	findings, err := tidbv1review.IndexPrimaryKeyTypeAllowlist(scriptNodes(stmts), stringArrayPayload.List, schemaOf(checkCtx.OriginalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.OriginalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexPKType.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", f.Name, f.Table, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(findingLine(stmts[f.Statement], f)),
		})
	}
	return adviceList, nil
}
