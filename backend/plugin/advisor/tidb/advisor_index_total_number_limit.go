package tidb

import (
	"context"
	"fmt"

	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor flags tables whose total index count (read
// from FinalMetadata) exceeds the configured maximum, gated on the reviewed
// statements touching the table with an index-creating operation. The
// judgment lives in omni (tidb/v1review.IndexTotalNumberLimit); this adapter
// passes the script and the simulated final schema and renders the findings
// as advice.
type IndexTotalNumberLimitAdvisor struct{}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for index total number limit rule")
	}
	maximum := int(numberPayload.Number)

	findings, err := tidbv1review.IndexTotalNumberLimit(scriptNodes(stmts), maximum, schemaOf(checkCtx.FinalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.FinalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexCountExceedsLimit.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %s", f.Table, maximum, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(findingLine(stmts[f.Statement], f)),
		})
	}
	return adviceList, nil
}
