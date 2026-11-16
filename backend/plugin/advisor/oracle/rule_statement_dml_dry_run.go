package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
}

// StatementDmlDryRunAdvisor is the advisor checking for DML dry run. omni
// (oracle/v1review.StatementDMLDryRunProbes) picks the statements to dry-run;
// this adapter runs EXPLAIN PLAN FOR them, and omni (StatementDMLDryRun)
// judges the results.
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
	if checkCtx.Driver == nil {
		return nil, nil
	}

	var nodes []ast.Node
	var stmts []base.ParsedStatement
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
	}

	probes := make(map[int]v1review.Probe)
	for n, i := range oraclev1review.StatementDMLDryRunProbes(nodes) {
		if n >= common.MaximumLintExplainSize {
			break
		}
		if _, err := advisor.Query(ctx, advisor.QueryContext{}, checkCtx.Driver, storepb.Engine_ORACLE, fmt.Sprintf("EXPLAIN PLAN FOR %s", stmts[i].Text)); err != nil {
			probes[i] = v1review.Probe{Error: err.Error()}
		} else {
			probes[i] = v1review.Probe{}
		}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range oraclev1review.StatementDMLDryRun(nodes, probes) {
		line := stmts[f.Statement].BaseLine() + 1
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementDMLDryRunFailed.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("Failed to dry run statement at line %d: %v", line, f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(line),
		})
	}
	return adviceList, nil
}
