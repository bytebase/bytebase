package mariadb

import (
	"context"
	"fmt"
	"strings"

	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
}

// StatementDmlDryRunAdvisor is the advisor checking for DML dry run. omni
// (mariadb/v1review.StatementDMLDryRunProbes) picks the statements to dry-run;
// this adapter EXPLAINs them, and omni (StatementDMLDryRun) judges the results.
type StatementDmlDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// BYT-8855: Skip DML dry run if there are DDL statements mixed in, because DML
	// statements often reference objects created by DDL statements, causing false positives.
	if advisor.ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
		return nil, nil
	}

	driver := checkCtx.Driver
	if driver == nil {
		return nil, nil
	}
	nodes, stmts := mariadbNodes(checkCtx.ParsedStatements)
	probes := make(map[int]v1review.Probe)
	for n, i := range mariadbv1review.StatementDMLDryRunProbes(nodes) {
		if _, err := advisor.Query(ctx, advisor.QueryContext{}, driver, storepb.Engine_MYSQL, fmt.Sprintf("EXPLAIN %s", dryRunText(stmts[i].Text))); err != nil {
			probes[i] = v1review.Probe{Error: err.Error()}
		} else {
			probes[i] = v1review.Probe{}
		}
		if n+1 >= common.MaximumLintExplainSize {
			break
		}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range mariadbv1review.StatementDMLDryRun(nodes, probes) {
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementDMLDryRunFailed.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", dryRunText(stmt.Text), f.Value),
			StartPosition: statementStartPosition(stmt),
		})
	}
	return adviceList, nil
}

// dryRunText is the statement as the dry run EXPLAINs and prints it: trimmed,
// with exactly one trailing semicolon.
func dryRunText(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ";") + ";"
}
