package tidb

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	// Cumulative #30 Codex-fix-2 (revised): resolve unqualified table
	// references to a default database at EXTRACTION time, so the
	// priorBackupTable.database field is always populated. This makes
	// the grouping key consistent across qualified vs unqualified
	// references to the same table. Use DBSchema.Name as the default
	// (the schema being checked; populated reliably across review
	// paths including plancheck where CurrentDatabase is not set —
	// per statement_advise_executor.go:168-180). Fall back to
	// CurrentDatabase, then empty string, in that order.
	defaultDB := ""
	if checkCtx.DBSchema != nil {
		defaultDB = checkCtx.DBSchema.GetName()
	}
	if defaultDB == "" {
		defaultDB = checkCtx.CurrentDatabase
	}
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
			// Cumulative #30 Codex-fix-1: derive UPDATE mutation
			// targets from SET-clause LHS qualifiers, NOT from the
			// full Tables list (which includes JOIN-only read-only
			// tables). For `UPDATE t1 JOIN t2 ON ... SET t1.col = ...`
			// the mutation target is t1; t2 must NOT be tagged.
			aliasMap := omniBuildTableAliasMap(n.Tables, defaultDB)
			for _, t := range omniExtractUpdateTargets(n.SetList, aliasMap) {
				dmlRefs = append(dmlRefs, dmlRef{table: t, stmtType: "UPDATE"})
			}
		case *ast.DeleteStmt:
			// DELETE's Tables field IS the mutation target set (per
			// omni parsenodes.go:123); Using[] is the filter-only
			// joins. Use Tables directly.
			for _, t := range omniExtractDMLTables(n.Tables, defaultDB) {
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
		// Cumulative #30 Codex-fix-2 (revised): database is already
		// resolved to defaultDB at extraction time (see omniExtractDMLTables
		// + omniBuildTableAliasMap), so equivalent references (qualified
		// vs unqualified) share the same database segment. Key is
		// lowercased for case-insensitive grouping.
		key := strings.ToLower(ref.table.database) + "." + strings.ToLower(ref.table.table)
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
// Unqualified references get defaultDB filled in (cumulative #30
// Codex-fix-2 revised: resolve at extraction time so grouping keys
// are consistent across qualified vs unqualified references).
// Cumulative #26 — UNION-rooted derived tables (accessed via
// SubqueryExpr at this layer) return nil. Derived tables aren't
// base-table candidates for backup-type mixing.
func omniExtractDMLTables(tables []ast.TableExpr, defaultDB string) []priorBackupTable {
	var result []priorBackupTable
	for _, t := range tables {
		result = append(result, omniExtractTableExprRefs(t, defaultDB)...)
	}
	return result
}

// omniExtractTableExprRefs walks a single omni TableExpr, returning
// the base-table references reached. Empty Schema is resolved to
// defaultDB (typically checkCtx.DBSchema.Name).
func omniExtractTableExprRefs(t ast.TableExpr, defaultDB string) []priorBackupTable {
	if t == nil {
		return nil
	}
	switch n := t.(type) {
	case *ast.TableRef:
		db := n.Schema
		if db == "" {
			db = defaultDB
		}
		return []priorBackupTable{{
			table:    n.Name,
			database: db,
		}}
	case *ast.JoinClause:
		left := omniExtractTableExprRefs(n.Left, defaultDB)
		right := omniExtractTableExprRefs(n.Right, defaultDB)
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

// updateTableAliasMap holds two pieces of resolution state for an
// UpdateStmt:
//   - lookup: alias-or-name → base table (used for qualified SET
//     column resolution)
//   - distinctBases: deduplicated base tables in the FROM clause
//     (used to detect single-target UPDATE for unqualified SET
//     attribution — pre-Codex-fix-1b counted aliasMap entries
//     directly, which double-counted aliased single-table UPDATEs
//     since each TableRef contributes 2 lookup keys)
type updateTableAliasMap struct {
	lookup        map[string]priorBackupTable
	distinctBases []priorBackupTable
}

// omniBuildTableAliasMap walks an UpdateStmt's Tables slice and
// builds resolution state for SET-clause target extraction. For each
// TableRef, registers BOTH the alias (if any) AND the bare name in
// the lookup map (so qualified references like `alias.col` or
// `tablename.col` resolve identically) and records the base table
// once in distinctBases. JoinClause recurses into Left+Right.
// SubqueryExpr contributes nothing — derived tables aren't base-
// table candidates (cumulative #26).
//
// Unqualified Schema is resolved to defaultDB (cumulative #30
// Codex-fix-2 revised).
//
// Pre-Codex-fix-1b regression: counted lookup-map entries (2 per
// aliased TableRef) when deciding "is this single-target?",
// dropping unqualified SET attribution for `UPDATE tech_book AS t
// SET id = 1`. Post-fix: count distinctBases instead.
func omniBuildTableAliasMap(tables []ast.TableExpr, defaultDB string) *updateTableAliasMap {
	m := &updateTableAliasMap{lookup: make(map[string]priorBackupTable)}
	for _, t := range tables {
		omniCollectTableAliases(t, defaultDB, m)
	}
	return m
}

func omniCollectTableAliases(t ast.TableExpr, defaultDB string, m *updateTableAliasMap) {
	if t == nil {
		return
	}
	switch n := t.(type) {
	case *ast.TableRef:
		db := n.Schema
		if db == "" {
			db = defaultDB
		}
		base := priorBackupTable{database: db, table: n.Name}
		if n.Alias != "" {
			m.lookup[strings.ToLower(n.Alias)] = base
		}
		m.lookup[strings.ToLower(n.Name)] = base
		// distinctBases dedup by canonicalized (db, table).
		distinctKey := strings.ToLower(base.database) + "." + strings.ToLower(base.table)
		alreadyTracked := false
		for _, b := range m.distinctBases {
			if strings.ToLower(b.database)+"."+strings.ToLower(b.table) == distinctKey {
				alreadyTracked = true
				break
			}
		}
		if !alreadyTracked {
			m.distinctBases = append(m.distinctBases, base)
		}
	case *ast.JoinClause:
		omniCollectTableAliases(n.Left, defaultDB, m)
		omniCollectTableAliases(n.Right, defaultDB, m)
	case *ast.SubqueryExpr:
		// Derived table — not a base-table candidate for backup
		// type-mixing detection.
	default:
	}
}

// omniExtractUpdateTargets walks an UpdateStmt's SetList and returns
// the distinct base-table references that are actual mutation
// targets. Resolves column qualifiers against the alias map:
//   - Qualified `t.col = ...` or `alias.col = ...`: look up the
//     qualifier in lookup → base table.
//   - Unqualified `col = ...`: if there is exactly ONE distinct
//     base table in the FROM clause (single-target UPDATE,
//     regardless of alias multiplicity), use it. Otherwise the
//     assignment is ambiguous without schema info — skip (multi-
//     target UPDATE with unqualified SET is rare in practice and
//     would be ambiguous-by-design).
//
// Returns deduplicated targets; duplicates from multiple assignments
// on the same table are collapsed.
func omniExtractUpdateTargets(setList []*ast.Assignment, m *updateTableAliasMap) []priorBackupTable {
	var result []priorBackupTable
	seen := make(map[string]bool)
	add := func(t priorBackupTable) {
		key := strings.ToLower(t.database) + "." + strings.ToLower(t.table)
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, t)
	}
	for _, a := range setList {
		if a == nil || a.Column == nil {
			continue
		}
		qualifier := a.Column.Table
		if qualifier == "" {
			// Unqualified: safe to attribute only when single-target.
			// Codex-fix-1b: count DISTINCT base tables, not lookup-
			// map entries — aliased TableRef contributes 2 lookup
			// keys but 1 distinct base.
			if len(m.distinctBases) == 1 {
				add(m.distinctBases[0])
			}
			continue
		}
		if base, ok := m.lookup[strings.ToLower(qualifier)]; ok {
			add(base)
		}
	}
	return result
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
