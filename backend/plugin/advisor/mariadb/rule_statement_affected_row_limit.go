package mariadb

import (
	"context"
	"fmt"

	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqldriver "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*StatementAffectedRowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, &StatementAffectedRowLimitAdvisor{})
}

// StatementAffectedRowLimitAdvisor is the advisor checking for UPDATE/DELETE
// affected row limit. omni (mariadb/v1review.StatementAffectedRowLimitProbes)
// picks the statements to estimate; this adapter EXPLAINs and estimates them,
// and omni (StatementAffectedRowLimit) judges the results.
type StatementAffectedRowLimitAdvisor struct {
}

// Check checks for UPDATE/DELETE affected row limit.
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
	driver := checkCtx.Driver
	if driver == nil {
		return nil, nil
	}

	title := checkCtx.Rule.Type.String()
	var explains advisor.ExplainBudget
	nodes, stmts := mariadbNodes(checkCtx.ParsedStatements)
	probes := make(map[int]v1review.Probe)
	for _, i := range mariadbv1review.StatementAffectedRowLimitProbes(nodes) {
		if !explains.Spend(statementStartPosition(stmts[i])) {
			continue
		}
		query := mariadbparser.AffectedRowsQuery(nodes[i], stmts[i].Text)
		plan, err := mysqldriver.ExplainJSON(ctx, driver, query)
		if err != nil {
			probes[i] = v1review.Probe{Error: err.Error()}
			continue
		}
		rowCount, err := mysqldriver.EstimateAffectedRowsMariaDB(ctx, driver, nodes[i], query, plan)
		if err != nil {
			probes[i] = v1review.Probe{CountError: err.Error()}
			continue
		}
		probes[i] = v1review.Probe{Rows: rowCount}
	}

	var advice []*storepb.Advice
	for _, f := range mariadbv1review.StatementAffectedRowLimit(nodes, probes, maxRow) {
		stmt := stmts[f.Statement]
		text := dryRunText(stmt.Text)
		adviceCode, content := code.StatementAffectedRowExceedsLimit, ""
		switch f.Code {
		case v1review.DryRunFailed:
			content = fmt.Sprintf("\"%s\" dry runs failed: %s", text, f.Value)
		case v1review.RowCountFailed:
			adviceCode, content = code.Internal, fmt.Sprintf("failed to get row count for \"%s\": %s", text, f.Value)
		case v1review.RowLimitExceeded:
			content = fmt.Sprintf("\"%s\" affected %s rows (estimated). The count exceeds %d.", text, f.Value, maxRow)
		default:
			continue
		}
		advice = append(advice, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         title,
			Content:       content,
			StartPosition: statementStartPosition(stmt),
		})
	}
	return explains.AppendSkippedAdvice(advice, title, code.StatementAffectedRowExceedsLimit), nil
}
