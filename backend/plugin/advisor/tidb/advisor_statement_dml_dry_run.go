package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
}

// StatementDmlDryRunAdvisor is the advisor checking for DML dry run. omni
// (tidb/v1review.StatementDMLDryRunProbes) picks the statements to dry-run;
// this adapter probes them, and omni (StatementDMLDryRun) judges the results.
type StatementDmlDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
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

	prober := &statementDmlDryRunProber{ctx: ctx, driver: checkCtx.Driver}
	nodes := make([]ast.Node, len(stmts))
	for i, stmt := range stmts {
		nodes[i] = stmt.Node
	}
	probes := make(map[int]v1review.Probe)
	for n, i := range tidbv1review.StatementDMLDryRunProbes(nodes) {
		if reason, failed := prober.probe(stmts[i]); failed {
			probes[i] = v1review.Probe{Error: reason}
		} else {
			probes[i] = v1review.Probe{}
		}
		if n+1 >= common.MaximumLintExplainSize {
			break
		}
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range tidbv1review.StatementDMLDryRun(nodes, probes) {
		ostmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.StatementDMLDryRunFailed.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", ostmt.TrimmedText(), f.Value),
			StartPosition: base.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
		})
	}
	return adviceList, nil
}

// statementDmlDryRunProber runs one statement's dry-run queries.
type statementDmlDryRunProber struct {
	ctx    context.Context
	driver *sql.DB
}

// probe dry-runs one statement and returns the reason it failed, if it did.
// A BATCH (non-transactional DML, e.g. "BATCH ON id LIMIT 5000 DELETE ..."):
//  1. Run "BATCH ... DRY RUN ..." to validate the batch-splitting logic.
//  2. EXPLAIN the inner DML to validate the DML itself.
func (c *statementDmlDryRunProber) probe(ostmt Stmt) (string, bool) {
	node, ok := ostmt.Node.(*ast.BatchStmt)
	if !ok {
		return c.explain(strings.TrimRight(ostmt.TrimmedText(), ";"))
	}
	// omni has no statement-level SQL deparse, so the DRY RUN form is derived by
	// injecting the keyword into the original statement text immediately before
	// the inner DML (located via the inner DML node's Loc). BATCH DRY RUN
	// requires auto-commit mode, so we use QueryContext directly instead of
	// advisor.Query (which wraps in a transaction).
	if dryRunSQL, ok := dmlDryRunBatchSQL(ostmt, node); ok {
		if err := c.queryInAutoCommit(dryRunSQL); err != nil {
			// Don't continue to EXPLAIN if BATCH DRY RUN failed.
			return err.Error(), true
		}
	}
	// EXPLAIN doesn't accept BATCH syntax, so we slice the inner DML out of the
	// original text via its Loc.
	innerSQL, ok := dmlDryRunInnerSQL(ostmt, node)
	if !ok {
		return "failed to extract inner DML", true
	}
	return c.explain(innerSQL)
}

// explain runs EXPLAIN on sql and returns the error's text if it failed.
func (c *statementDmlDryRunProber) explain(sql string) (string, bool) {
	if _, err := advisor.Query(c.ctx, advisor.QueryContext{}, c.driver, storepb.Engine_TIDB, fmt.Sprintf("EXPLAIN %s", sql)); err != nil {
		return err.Error(), true
	}
	return "", false
}

// queryInAutoCommit runs a query in auto-commit mode (without wrapping in a
// transaction). Required for TiDB's BATCH DRY RUN, which cannot run inside a
// transaction.
func (c *statementDmlDryRunProber) queryInAutoCommit(statement string) error {
	rows, err := c.driver.QueryContext(c.ctx, statement)
	if err != nil {
		return err
	}
	defer rows.Close()
	// Drain the rows to ensure the query completes.
	for rows.Next() {
	}
	return rows.Err()
}

// dmlDryRunBatchSQL returns a "BATCH ... DRY RUN ..." statement for node.
//
// omni has no statement-level SQL deparse, so the DRY RUN form is derived by
// injecting "DRY RUN " before the inner DML keyword (located via the inner DML
// node's Loc), preserving the user's original ON/LIMIT clause, comments, and
// whitespace verbatim — more faithful than a deparser.
//
// "DRY RUN" selects TiDB's split-DML dry run (pingcap DryRunSplitDml=2), which
// validates the generated split DMLs — the batch-splitting logic this advisor
// exists to check. The pre-omni advisor set DryRun=1, i.e. pingcap's
// DryRunQuery ("DRY RUN QUERY"), which validates only the shard-splitting
// SELECT, not the split DMLs — contradicting its own comment. Container-
// verified: "DRY RUN QUERY" passes a bad inner-DML WHERE that "DRY RUN" rejects
// (the advisor's EXPLAIN step also rejects it, so the net advice is unchanged).
//
// If the source already specifies a dry-run mode, it is run as-is; injecting
// again would yield "DRY RUN DRY RUN".
func dmlDryRunBatchSQL(ostmt Stmt, node *ast.BatchStmt) (string, bool) {
	if node.DryRun != ast.BatchDryRunNone {
		return ostmt.Text, true
	}
	loc, ok := dmlDryRunInnerLoc(node.DML)
	if !ok || loc.Start <= 0 || loc.Start > len(ostmt.Text) {
		return "", false
	}
	return ostmt.Text[:loc.Start] + "DRY RUN " + ostmt.Text[loc.Start:], true
}

// dmlDryRunInnerSQL slices the inner DML out of the original BATCH statement
// text via the inner DML node's Loc (statement-relative byte offsets).
func dmlDryRunInnerSQL(ostmt Stmt, node *ast.BatchStmt) (string, bool) {
	loc, ok := dmlDryRunInnerLoc(node.DML)
	if !ok || loc.Start <= 0 || loc.Start > len(ostmt.Text) {
		return "", false
	}
	end := loc.End
	if end <= loc.Start || end > len(ostmt.Text) {
		end = len(ostmt.Text)
	}
	return strings.TrimRight(strings.TrimSpace(ostmt.Text[loc.Start:end]), ";"), true
}

// dmlDryRunInnerLoc returns the source location of a BATCH inner DML node. The
// field is interface-typed (ast.StmtNode), concretely one of the three DML
// node types omni produces under BATCH (REPLACE is *InsertStmt{IsReplace}).
func dmlDryRunInnerLoc(n ast.StmtNode) (ast.Loc, bool) {
	switch s := n.(type) {
	case *ast.DeleteStmt:
		return s.Loc, true
	case *ast.UpdateStmt:
		return s.Loc, true
	case *ast.InsertStmt:
		return s.Loc, true
	default:
		return ast.Loc{}, false
	}
}
