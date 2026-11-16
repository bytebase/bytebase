package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor flags index/constraint declarations whose
// number of key columns exceeds the configured maximum. The judgment lives
// in omni (tidb/v1review.IndexKeyNumberLimit); this adapter converts the
// rule's payload and renders the findings as advice.
type IndexKeyNumberLimitAdvisor struct{}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		return nil, errors.New("number_payload is required for index key number limit rule")
	}
	maximum := int(numberPayload.Number)
	if maximum <= 0 {
		return nil, nil
	}
	title := checkCtx.Rule.Type.String()

	adviceList := []*storepb.Advice{}
	for _, stmt := range stmts {
		for _, f := range tidbv1review.IndexKeyNumberLimit([]ast.Node{stmt.Node}, maximum) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.IndexKeyNumberExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The number of index `%s` in table `%s` should be not greater than %d", f.Name, f.Table, maximum),
				StartPosition: base.ConvertANTLRLineToPosition(findingLine(stmt, f)),
			})
		}
	}
	return adviceList, nil
}
