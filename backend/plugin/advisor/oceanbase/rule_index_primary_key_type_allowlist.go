package oceanbase

import (
	"context"
	"fmt"

	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*IndexPrimaryKeyTypeAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST, &IndexPrimaryKeyTypeAllowlistAdvisor{})
}

// IndexPrimaryKeyTypeAllowlistAdvisor is the advisor checking for primary key
// type allowlist. The judgment lives in omni
// (oceanbase/v1review.IndexPrimaryKeyTypeAllowlist); this adapter passes the
// script, the payload and the pre-change schema and renders the findings as
// advice.
type IndexPrimaryKeyTypeAllowlistAdvisor struct {
}

// Check checks for primary key type allowlist.
func (*IndexPrimaryKeyTypeAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// A rule without the payload panics here, which advisor.Check reports
	// as the rule's error.
	allowlist := checkCtx.Rule.GetStringArrayPayload().List

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	findings, err := oceanbasev1review.IndexPrimaryKeyTypeAllowlist(nodes, allowlist, schemaOf(checkCtx.OriginalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.OriginalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexPKType.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", f.Name, f.Table, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
		})
	}
	return adviceList, nil
}
