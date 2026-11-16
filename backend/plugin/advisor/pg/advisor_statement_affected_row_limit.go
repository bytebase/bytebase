package pg

import (
	"context"
	"fmt"

	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE/MERGE affected row limit.
// omni (pg/v1review.StatementAffectedRowLimitProbes) picks the statements to
// estimate; this adapter EXPLAINs them under the session settings before
// them, and omni (StatementAffectedRowLimit) judges the results.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE/MERGE affected row limit, including data-modifying CTEs.
func (*StatementAffectedRowLimitAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}
	maxRow := int(numberPayload.Number)
	if maxRow <= 0 || checkCtx.Driver == nil {
		return nil, nil
	}

	title := checkCtx.Rule.Type.String()
	var explains advisor.ExplainBudget
	l := newLiveStatements(checkCtx.ParsedStatements)
	probes := make(map[int]v1review.Probe)
	l.probeEach(pgv1review.StatementAffectedRowLimitProbes(l.nodes), func(i int, settings []string) {
		if !explains.Spend(&storepb.Position{Line: l.startLine(i) + int32(l.stmts[i].BaseLine())}) {
			return
		}
		probes[i] = explainRows(ctx, checkCtx.Driver, checkCtx.TenantMode, settings, l.probeText(i), getAffectedRows)
	})

	var adviceList []*storepb.Advice
	for _, f := range pgv1review.StatementAffectedRowLimit(l.nodes, probes, maxRow) {
		text := l.probeText(f.Statement)
		adviceCode, content := code.StatementAffectedRowExceedsLimit, ""
		switch f.Code {
		case v1review.DryRunFailed:
			content = fmt.Sprintf("\"%s\" dry runs failed: %s", text, f.Value)
		case v1review.RowCountFailed:
			adviceCode, content = code.Internal, fmt.Sprintf("failed to get row count for \"%s\": %s", text, f.Value)
		case v1review.RowLimitExceeded:
			content = fmt.Sprintf("The statement \"%s\" affected %s rows (estimated). The count exceeds %d.", text, f.Value, maxRow)
		default:
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    adviceCode.Int32(),
			Title:   title,
			Content: content,
			StartPosition: &storepb.Position{
				Line:   l.startLine(f.Statement) + int32(l.stmts[f.Statement].BaseLine()),
				Column: 0,
			},
		})
	}
	return explains.AppendSkippedAdvice(adviceList, title, code.StatementAffectedRowExceedsLimit), nil
}
