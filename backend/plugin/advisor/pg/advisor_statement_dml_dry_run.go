package pg

import (
	"context"
	"fmt"

	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDMLDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDMLDryRunAdvisor{})
}

// StatementDMLDryRunAdvisor is the advisor checking for DML dry run. omni
// (pg/v1review.StatementDMLDryRunProbes) picks the statements to dry-run;
// this adapter EXPLAINs them under the session settings before them, and
// omni (StatementDMLDryRun) judges the results.
type StatementDMLDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDMLDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	// Only run EXPLAIN queries if we have a database connection
	if checkCtx.Driver == nil {
		return nil, nil
	}
	// BYT-8855: Skip DML dry run if there are DDL statements mixed in, because DML
	// statements often reference objects created by DDL statements, causing false positives.
	if advisor.ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
		return nil, nil
	}

	l := newLiveStatements(checkCtx.ParsedStatements)
	probes := make(map[int]v1review.Probe)
	explained := 0
	l.probeEach(pgv1review.StatementDMLDryRunProbes(l.nodes), func(i int, settings []string) {
		if explained >= common.MaximumLintExplainSize {
			return
		}
		explained++
		if _, err := advisor.Query(ctx, advisor.QueryContext{
			TenantMode:    checkCtx.TenantMode,
			PreExecutions: settings,
		}, checkCtx.Driver, storepb.Engine_POSTGRES, fmt.Sprintf("EXPLAIN %s", l.probeText(i))); err != nil {
			probes[i] = v1review.Probe{Error: err.Error()}
		} else {
			probes[i] = v1review.Probe{}
		}
	})

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range pgv1review.StatementDMLDryRun(l.nodes, probes) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    code.StatementDMLDryRunFailed.Int32(),
			Title:   title,
			Content: fmt.Sprintf("\"%s\" dry runs failed: %s", l.probeText(f.Statement), f.Value),
			StartPosition: &storepb.Position{
				Line:   l.startLine(f.Statement) + int32(l.stmts[f.Statement].BaseLine()),
				Column: 0,
			},
		})
	}
	return adviceList, nil
}
