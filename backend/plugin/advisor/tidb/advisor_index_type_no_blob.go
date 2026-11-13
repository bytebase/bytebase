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
	_ advisor.Advisor = (*IndexTypeNoBlobAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB, &IndexTypeNoBlobAdvisor{})
}

// IndexTypeNoBlobAdvisor is the advisor checking for index type no blob. The
// judgment lives in omni (tidb/v1review.IndexTypeNoBlob); this adapter
// passes the script and the pre-change schema and renders the findings as
// advice.
type IndexTypeNoBlobAdvisor struct {
}

// Check checks for index type no blob.
func (*IndexTypeNoBlobAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	findings, err := tidbv1review.IndexTypeNoBlob(scriptNodes(stmts), schemaOf(checkCtx.OriginalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.OriginalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexTypeNoBlob.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", f.Table, f.Name, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(findingLine(stmts[f.Statement], f)),
		})
	}
	return adviceList, nil
}
