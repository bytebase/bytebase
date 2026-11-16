package oceanbase

import (
	"context"
	"fmt"

	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
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
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number
// limit. The judgment lives in omni (oceanbase/v1review.IndexTotalNumberLimit);
// this adapter passes the script and the simulated final schema and renders
// the findings as advice.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	limit := int(numberPayload.Number)

	stmts, nodes := scriptNodes(checkCtx.ParsedStatements)
	findings, err := oceanbasev1review.IndexTotalNumberLimit(nodes, limit, schemaOf(checkCtx.FinalMetadata))
	if err != nil {
		readMissingSchema(checkCtx.FinalMetadata, err)
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range findings {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.IndexCountExceedsLimit.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %s", f.Table, limit, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
		})
	}
	return adviceList, nil
}
