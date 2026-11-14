package oceanbase

import (
	"context"
	"fmt"

	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*InsertRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, &InsertRowLimitAdvisor{})
}

// InsertRowLimitAdvisor is the advisor checking for insert row limit. omni
// (oceanbase/v1review.StatementInsertRowLimitProbes) picks the INSERTs whose
// rows the database must estimate; this adapter EXPLAINs them, and omni
// (StatementInsertRowLimit) judges every INSERT.
type InsertRowLimitAdvisor struct {
}

// Check checks for insert row limit.
func (*InsertRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	maxRow := int(numberPayload.Number)
	title := checkCtx.Rule.Type.String()

	var explains advisor.ExplainBudget
	nodes, stmts := oceanbaseNodes(checkCtx.ParsedStatements)
	probes := make(map[int]v1review.Probe)
	if checkCtx.Driver != nil {
		for _, i := range oceanbasev1review.StatementInsertRowLimitProbes(nodes) {
			if !explains.Spend(nodePosition(stmts[i], nodes[i])) {
				continue
			}
			probes[i] = explainRows(ctx, checkCtx.Driver, stmts[i].Text)
		}
	}

	var adviceList []*storepb.Advice
	for _, f := range oceanbasev1review.StatementInsertRowLimit(nodes, probes, maxRow) {
		text := stmts[f.Statement].Text
		adviceCode, content := code.InsertTooManyRows, ""
		switch f.Code {
		case v1review.DryRunFailed:
			content = fmt.Sprintf("\"%s\" dry runs failed: %s", text, f.Value)
		case v1review.RowCountFailed:
			adviceCode, content = code.Internal, fmt.Sprintf("failed to get row count for \"%s\": %s", text, f.Value)
		case v1review.RowLimitExceeded:
			content = fmt.Sprintf("\"%s\" inserts %s rows. The count exceeds %d.", text, f.Value, maxRow)
		default:
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         title,
			Content:       content,
			StartPosition: findingPosition(stmts[f.Statement], f),
		})
	}
	return explains.AppendSkippedAdvice(adviceList, title, code.InsertTooManyRows), nil
}
