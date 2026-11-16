package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor is the advisor checking for index key number
// limit. The judgment lives in omni (pg/v1review.IndexKeyNumberLimit); this
// adapter converts the rule's payload and renders the findings as advice.
type IndexKeyNumberLimitAdvisor struct {
}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for i, node := range nodes {
		for _, f := range pgv1review.IndexKeyNumberLimit([]ast.Node{node}, limit) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.IndexKeyNumberExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", f.Name, f.Table, limit),
				StartPosition: findingPosition(stmts[i], f),
			})
		}
	}
	return adviceList, nil
}
