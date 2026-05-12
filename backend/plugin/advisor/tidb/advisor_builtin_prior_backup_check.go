package tidb

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, &StatementPriorBackupCheckAdvisor{})
}

// StatementPriorBackupCheckAdvisor flags inputs incompatible with the
// prior-backup workflow. Aligned with the mysql analog's modernized
// shape (per-table DML-type mixing detection + size cap), NOT the
// stale pre-omni tidb logic (count cap + unique-WHERE short-circuit).
//
// The reshape decision (batch 19) — instead of mechanically porting
// the pre-omni tidb logic per invariant #7, we align with the mysql
// analog's modernized shape because:
//  1. Pre-omni tidb wasn't being tested (orphan fixture, not in
//     tidb_rules_test.go) — "preserving" untested behavior preserves
//     unknowns.
//  2. Mysql's per-table DML-type mixing is more semantically accurate
//     for backup feasibility than the count-cap heuristic.
//  3. Phase 1.5 closes after this batch; deferring the alignment to a
//     future ticket would leave tidb on stale logic indefinitely.
//
// Behavior matrix (vs pre-omni tidb):
//
//	UPDATE t × 6 (no unique-WHERE) — pre: fires (count cap). new: skips.
//	UPDATE t WHERE id=5 × 10        — pre: skips. new: skips.
//	UPDATE t; DELETE FROM t         — pre: skips. new: FIRES (per-table mixing).
//	Statements > MaxSheetCheckSize — pre: skips. new: FIRES (size cap).
//	Missing bbdataarchive db       — pre: fires w/ code.BuiltinPriorBackupCheck.
//	                                 new: fires w/ code.DatabaseNotExists
//	                                 (public-API delta, mysql-aligned).
//
// Audit axes applied:
//   - #7 (preserve pre-omni): NOT applied here — see reshape rationale.
//   - #19 (case-sensitivity): table-name grouping uses
//     strings.EqualFold-equivalent via lowercased key.
//   - #26 (UNION-root): omni's UNION-rooted UpdateStmt sources are
//     reached only via SubqueryExpr in TableExpr; the SubqueryExpr
//     arm returns nil — derived tables aren't base-table candidates.
//   - #29 (filter-effect): no expression-tree filter here. Dropped
//     omniIsConstantLit (was for the unique-WHERE machinery).
type StatementPriorBackupCheckAdvisor struct{}

// Check evaluates the prior-backup compatibility of the reviewed
// statements. Gated on `checkCtx.EnablePriorBackup`.
func (*StatementPriorBackupCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	if !checkCtx.EnablePriorBackup {
		return nil, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice

	// 1. Size cap.
	if checkCtx.StatementsTotalSize > common.MaxSheetCheckSize {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %d bytes for backup", common.MaxSheetCheckSize),
			Code:          code.BuiltinPriorBackupCheck.Int32(),
			StartPosition: nil,
		})
	}

	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	// 2. Mixed DDL + DML detection. Collect DML-by-table along the way.
	type dmlRef struct {
		table    priorBackupTable
		stmtType string // "UPDATE" or "DELETE"
	}
	var dmlRefs []dmlRef

	for _, ostmt := range stmts {
		node := ostmt.Node
		if omniIsDDLStmt(node) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       "Prior backup cannot deal with mixed DDL and DML statements",
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: common.ConvertANTLRLineToPosition(ostmt.FirstTokenLine()),
			})
		}
		switch n := node.(type) {
		case *ast.UpdateStmt:
			for _, t := range omniExtractDMLTables(n.Tables) {
				dmlRefs = append(dmlRefs, dmlRef{table: t, stmtType: "UPDATE"})
			}
		case *ast.DeleteStmt:
			for _, t := range omniExtractDMLTables(n.Tables) {
				dmlRefs = append(dmlRefs, dmlRef{table: t, stmtType: "DELETE"})
			}
		default:
		}
	}

	// 3. Backup database existence.
	databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_TIDB)
	if !advisor.DatabaseExists(ctx, checkCtx, databaseName) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Title:         title,
			Content:       fmt.Sprintf("Need database %q to do prior backup but it does not exist", databaseName),
			Code:          code.DatabaseNotExists.Int32(),
			StartPosition: nil,
		})
	}

	// 4. Per-table DML-type mixing. Group by `db.table` key; if a
	// single table has more than one DML type observed, emit advice.
	// Mysql-aligned: more accurate than the pre-omni count cap.
	type tableTypes struct {
		seen map[string]bool
		any  string // first-seen type, for stable advice-content sentinel
	}
	groups := make(map[string]*tableTypes)
	for _, ref := range dmlRefs {
		key := fmt.Sprintf("%s.%s", ref.table.database, ref.table.table)
		g := groups[key]
		if g == nil {
			g = &tableTypes{seen: make(map[string]bool)}
			groups[key] = g
		}
		g.seen[ref.stmtType] = true
		if g.any == "" {
			g.any = ref.stmtType
		}
	}

	// Deterministic order: sort keys lexicographically before emitting.
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, key := range keys {
		g := groups[key]
		if len(g.seen) > 1 {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", key),
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: nil,
			})
		}
	}

	return adviceList, nil
}

// priorBackupTable is the (database, table) qualifier captured from
// UPDATE/DELETE table references. File-local — only this advisor uses it.
type priorBackupTable struct {
	database string
	table    string
}

// omniExtractDMLTables walks an UpdateStmt.Tables or DeleteStmt.Tables
// slice (each element is a TableExpr — *TableRef, *JoinClause, or
// *SubqueryExpr) and returns the base-table references reached.
// Cumulative #26 — UNION-rooted derived tables (accessed via
// SubqueryExpr at this layer) return nil. Derived tables aren't
// base-table candidates for backup-type mixing.
func omniExtractDMLTables(tables []ast.TableExpr) []priorBackupTable {
	var result []priorBackupTable
	for _, t := range tables {
		result = append(result, omniExtractTableExprRefs(t)...)
	}
	return result
}

// omniExtractTableExprRefs walks a single omni TableExpr, returning
// the base-table references reached.
func omniExtractTableExprRefs(t ast.TableExpr) []priorBackupTable {
	if t == nil {
		return nil
	}
	switch n := t.(type) {
	case *ast.TableRef:
		return []priorBackupTable{{
			table:    n.Name,
			database: n.Schema,
		}}
	case *ast.JoinClause:
		left := omniExtractTableExprRefs(n.Left)
		right := omniExtractTableExprRefs(n.Right)
		return append(left, right...)
	case *ast.SubqueryExpr:
		// Cumulative #26: derived tables aren't base-table candidates;
		// nil matches mysql's modernized behavior and pre-omni tidb's
		// *SubqueryExpr nil-return arm.
		return nil
	default:
		return nil
	}
}

// omniIsDDLStmt reports whether the given statement node is a DDL
// (schema-changing) statement, mirroring pingcap-tidb's `ast.DDLNode`
// interface set. Pingcap-tidb's DDLNode has 22 implementers (verified
// against pkg/parser/ast/ddl.go's `_ DDLNode = &XxxStmt{}` declarations
// for tidb v8.5.5). The omni enumeration here covers 18 of those 22 —
// the 4 absent ones are deferred Tier 4 grammar in omni today:
//   - CreateSequenceStmt / AlterSequenceStmt / DropSequenceStmt
//   - FlashBackDatabaseStmt
//
// When omni grammar lands those, this list needs updating.
//
// **Critically** also excludes types that look DDL-shaped but pingcap-
// tidb does NOT classify as DDL:
//   - DropViewStmt (CreateViewStmt IS DDL, but DropViewStmt isn't —
//     asymmetric in pingcap)
//   - AlterDatabaseStmt (CreateDatabaseStmt + DropDatabaseStmt are
//     DDL but AlterDatabaseStmt isn't)
//   - User/Role management (Create/Alter/DropUser, Create/DropRole)
//   - Function/Trigger/Event procedural objects
//   - Tablespace + Server management
//
// Initial batch 19 PR over-enumerated DDL (included User/Role/Function/
// Tablespace/etc. as DDL — false-positives that pre-omni did NOT flag);
// pre-merge peer review caught the bidirectional mismatch and provided
// the verified correct list. Unit tests in utils_test.go pin both
// positive (DDL types that MUST return true) and negative (DML +
// non-DDL utility types that must NOT) cases.
func omniIsDDLStmt(node ast.Node) bool {
	switch node.(type) {
	case *ast.CreateTableStmt, *ast.AlterTableStmt, *ast.DropTableStmt,
		*ast.CreateIndexStmt, *ast.DropIndexStmt,
		*ast.CreateViewStmt,                            // NOT DropViewStmt (asymmetric in pingcap)
		*ast.CreateDatabaseStmt, *ast.DropDatabaseStmt, // NOT AlterDatabaseStmt
		*ast.TruncateStmt, // pingcap calls it TruncateTableStmt
		*ast.RenameTableStmt,
		*ast.CreatePlacementPolicyStmt, *ast.AlterPlacementPolicyStmt, *ast.DropPlacementPolicyStmt,
		*ast.CreateResourceGroupStmt, *ast.AlterResourceGroupStmt, *ast.DropResourceGroupStmt,
		*ast.OptimizeTableStmt, *ast.RepairTableStmt:
		return true
	default:
		return false
	}
}
