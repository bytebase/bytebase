package tidb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementDmlDryRunAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, &StatementDmlDryRunAdvisor{})
}

// StatementDmlDryRunAdvisor is the advisor checking for DML dry run.
type StatementDmlDryRunAdvisor struct {
}

// Check checks for DML dry run.
func (*StatementDmlDryRunAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
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

	checker := &statementDmlDryRunChecker{
		ctx:    ctx,
		driver: checkCtx.Driver,
		level:  level,
		title:  checkCtx.Rule.Type.String(),
	}
	for _, ostmt := range stmts {
		checker.check(ostmt)
		if checker.explainCount >= common.MaximumLintExplainSize {
			break
		}
	}

	return checker.adviceList, nil
}

type statementDmlDryRunChecker struct {
	ctx          context.Context
	driver       *sql.DB
	level        storepb.Advice_Status
	title        string
	adviceList   []*storepb.Advice
	explainCount int
}

// check dispatches a single top-level statement. Recipe A (top-level
// type-switch): omni surfaces TiDB BATCH as *ast.BatchStmt and plain DML as
// *ast.{Insert,Update,Delete}Stmt — both top-level here, so a walk is
// unnecessary. EXPLAIN-wrapped DML (e.g. "EXPLAIN DELETE ...") is degenerate
// input for a dry-run advisor and intentionally not descended into.
func (c *statementDmlDryRunChecker) check(ostmt OmniStmt) {
	switch node := ostmt.Node.(type) {
	case *ast.BatchStmt:
		c.checkBatch(ostmt, node)
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
		c.explainCount++
		c.explain(ostmt, strings.TrimRight(ostmt.TrimmedText(), ";"))
	default:
	}
}

// checkBatch validates a TiDB BATCH (non-transactional DML) statement, e.g.
// "BATCH ON id LIMIT 5000 DELETE ...":
//  1. Run "BATCH ... DRY RUN ..." to validate the batch-splitting logic.
//  2. EXPLAIN the inner DML to validate the DML itself.
func (c *statementDmlDryRunChecker) checkBatch(ostmt OmniStmt, node *ast.BatchStmt) {
	c.explainCount++

	// Step 1: validate batch splitting via TiDB's native BATCH DRY RUN. omni
	// has no statement-level SQL deparse, so the DRY RUN form is derived by
	// injecting the keyword into the original statement text immediately before
	// the inner DML (located via the inner DML node's Loc). BATCH DRY RUN
	// requires auto-commit mode, so we use QueryContext directly instead of
	// advisor.Query (which wraps in a transaction).
	if dryRunSQL, ok := dmlDryRunBatchSQL(ostmt, node); ok {
		if err := c.queryInAutoCommit(dryRunSQL); err != nil {
			c.appendFailure(ostmt, err.Error())
			// Don't continue to EXPLAIN if BATCH DRY RUN failed.
			return
		}
	}

	// Step 2: EXPLAIN the inner DML. EXPLAIN doesn't accept BATCH syntax, so we
	// slice the inner DML out of the original text via its Loc.
	innerSQL, ok := dmlDryRunInnerSQL(ostmt, node)
	if !ok {
		c.appendFailure(ostmt, "failed to extract inner DML")
		return
	}
	c.explain(ostmt, innerSQL)
}

// explain runs EXPLAIN on sql and records advice if it fails. The advice text
// is the full original statement (ostmt), matching the pre-omni behavior where
// BATCH advices reported the whole BATCH statement, not the inner DML.
func (c *statementDmlDryRunChecker) explain(ostmt OmniStmt, sql string) {
	if _, err := advisor.Query(c.ctx, advisor.QueryContext{}, c.driver, storepb.Engine_TIDB, fmt.Sprintf("EXPLAIN %s", sql)); err != nil {
		c.appendFailure(ostmt, err.Error())
	}
}

func (c *statementDmlDryRunChecker) appendFailure(ostmt OmniStmt, reason string) {
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:        c.level,
		Code:          code.StatementDMLDryRunFailed.Int32(),
		Title:         c.title,
		Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", ostmt.TrimmedText(), reason),
		StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
	})
}

// queryInAutoCommit runs a query in auto-commit mode (without wrapping in a
// transaction). Required for TiDB's BATCH DRY RUN, which cannot run inside a
// transaction.
func (c *statementDmlDryRunChecker) queryInAutoCommit(statement string) error {
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
func dmlDryRunBatchSQL(ostmt OmniStmt, node *ast.BatchStmt) (string, bool) {
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
func dmlDryRunInnerSQL(ostmt OmniStmt, node *ast.BatchStmt) (string, bool) {
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
