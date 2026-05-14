package tidb

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	pingcapast "github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementPriorBackupCheckAdvisor)(nil)
)

// maxMixedDMLCount must stay in sync with the backup transformer's
// constant at backend/plugin/parser/tidb/backup.go:23. The advisor's
// count-cap gate (Codex-fix-1g) uses the same threshold to predict
// transformer behavior at pre-execution time.
const maxMixedDMLCount = 5

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

	// Cumulative #30 Codex-fix-1f: DDL detection MUST go through
	// pingcap (`getTiDBNodes`), not omni (`getTiDBOmniNodes`).
	// Reason: invariant #2 soft-fail at the omni wrapper silently
	// drops Tier-4-deferred grammar (Sequence trio, FlashBackDatabase,
	// FlashBackTable). For a safety gate, that's a real regression —
	// pingcap's full DDLNode interface catches all 22 implementers
	// via `ddlNode` struct embedding. The omni path is still used
	// below for per-table DML-mixing detection (where omni AST shape
	// is required); DDL detection sidesteps omni's grammar gaps via
	// the pingcap-bridge path.
	//
	// Architectural note: this dual-path approach mirrors mysql's
	// analog (mysqlparser.ExtractTables + isOmniDDL). Eight rounds
	// of Codex catches on omni-only DDL enumeration (cumulative #30
	// Codex-fix-1c through 1e) demonstrated the brittle-enumeration
	// pattern; switching DDL detection to pingcap's authoritative
	// DDLNode interface eliminates the enumeration entirely.
	pingcapStmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	for _, p := range pingcapStmts {
		if _, isDDL := p.(pingcapast.DDLNode); isDDL {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       "Prior backup cannot deal with mixed DDL and DML statements",
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: common.ConvertANTLRLineToPosition(p.OriginTextPosition()),
			})
		}
	}

	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	// 2. Per-table DML mixing detection (omni path — needs full
	// AST-shape access for SET-clause analysis).
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
		switch n := node.(type) {
		case *ast.UpdateStmt:
			// Cumulative #30 Codex-fix-1: derive UPDATE mutation
			// targets from SET-clause LHS qualifiers, NOT from the
			// full Tables list (which includes JOIN-only read-only
			// tables). For `UPDATE t1 JOIN t2 ON ... SET t1.col = ...`
			// the mutation target is t1; t2 must NOT be tagged.
			aliasMap := omniBuildTableAliasMap(n.Tables, defaultDB)
			for _, t := range omniExtractUpdateTargets(n.SetList, aliasMap, checkCtx.DBSchema) {
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

	// Cumulative #30 Codex-fix-1g: count-cap with multi-table gate.
	// The backup transformer at backend/plugin/parser/tidb/backup.go:
	// 96-110 routes > maxMixedDMLCount DML statements into
	// generateSQLForSingleTable which errors on multi-table inputs
	// ("prior backup cannot handle statements on different tables
	// more than 5"). My reshape (cumulative #30) dropped this gate
	// under the "modernize-away-pre-omni-logic" framing — but the
	// transformer constraint is current, not legacy. Reinstating
	// the count gate prevents the advisor from approving inputs
	// that the transformer rejects at runtime.
	//
	// Single-table batches above the threshold are intentionally
	// allowed — the transformer's generateSQLForSingleTable
	// successfully handles them.
	if len(dmlRefs) > maxMixedDMLCount {
		distinctDMLTables := make(map[string]struct{})
		for _, ref := range dmlRefs {
			distinctKey := strings.ToLower(ref.table.database) + "." + strings.ToLower(ref.table.table)
			distinctDMLTables[distinctKey] = struct{}{}
		}
		if len(distinctDMLTables) > 1 {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       fmt.Sprintf("Prior backup cannot handle more than %d DML statements across different tables", maxMixedDMLCount),
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

// updateTableAliasMap holds resolution state for an UpdateStmt's
// SET-clause target extraction. Three lookup paths plus a single-
// target-detection list:
//
//   - bySchemaName: canonical "schema.name" key → base. Used for
//     fully-qualified SET LHS (`SET db1.t.col = ...`). Cumulative
//     #30 Codex-fix-1d: required because joined tables with the
//     same bare name in different schemas (e.g.
//     `UPDATE db1.tech_book JOIN db2.tech_book ...`) collide in a
//     bare-name-only lookup. Schema-qualified lookup disambiguates.
//
//   - byAlias: canonical alias key → base. Only registered for
//     aliased TableRefs. Aliases are unique within a query (parser-
//     enforced) so no ambiguity.
//
//   - byBareName: canonical bare-name key → list of bases. Multiple
//     bases under the same bare name means the bare-name reference
//     is ambiguous (resolution skips to avoid misattribution).
//
//   - distinctBases: deduplicated base tables in the FROM clause.
//     Used to detect single-target UPDATE for unqualified-column
//     SET attribution. Pre-Codex-fix-1b counted lookup-map entries
//     directly, which double-counted aliased TableRefs.
type updateTableAliasMap struct {
	bySchemaName  map[string]priorBackupTable
	byAlias       map[string]priorBackupTable
	byBareName    map[string][]priorBackupTable
	distinctBases []priorBackupTable
}

// omniBuildTableAliasMap walks an UpdateStmt's Tables slice and
// builds resolution state for SET-clause target extraction.
// Unqualified TableRef Schema is resolved to defaultDB.
// JoinClause recurses into Left+Right. SubqueryExpr contributes
// nothing — derived tables aren't base-table candidates (cumulative
// #26).
func omniBuildTableAliasMap(tables []ast.TableExpr, defaultDB string) *updateTableAliasMap {
	m := &updateTableAliasMap{
		bySchemaName: make(map[string]priorBackupTable),
		byAlias:      make(map[string]priorBackupTable),
		byBareName:   make(map[string][]priorBackupTable),
	}
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
		nameLower := strings.ToLower(n.Name)
		dbLower := strings.ToLower(db)
		m.bySchemaName[dbLower+"."+nameLower] = base
		if n.Alias != "" {
			m.byAlias[strings.ToLower(n.Alias)] = base
		}
		m.byBareName[nameLower] = append(m.byBareName[nameLower], base)
		// distinctBases dedup by canonicalized (db, table).
		distinctKey := dbLower + "." + nameLower
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
		// Derived table — not a base-table candidate.
	default:
	}
}

// omniExtractUpdateTargets walks an UpdateStmt's SetList and returns
// the distinct base-table references that are actual mutation
// targets. Resolution by qualifier type:
//
//  1. Column.Schema != "" AND Column.Table != "" → fully-qualified
//     lookup via bySchemaName. Cumulative #30 Codex-fix-1d.
//  2. Column.Schema == "" AND Column.Table != "" → byAlias first
//     (aliases unambiguous); on miss, byBareName. If byBareName
//     has multiple candidates (joined same-named tables across
//     schemas), the reference is ambiguous → skip.
//  3. Both empty → single-target shortcut via distinctBases.
//     Multi-target falls through to schema-aware column resolution
//     (Codex-fix-1e): walk dbMetadata for each candidate base to
//     find which one owns the column; single match → use; multiple
//     or zero → skip (MySQL itself errors on ambiguous unqualified
//     refs at execution time).
//
// Returns deduplicated targets.
func omniExtractUpdateTargets(setList []*ast.Assignment, m *updateTableAliasMap, dbMetadata *storepb.DatabaseSchemaMetadata) []priorBackupTable {
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
		col := a.Column
		switch {
		case col.Schema != "" && col.Table != "":
			// Cumulative #30 Codex-fix-1d: schema-qualified lookup
			// disambiguates same-bare-name joined tables across schemas.
			key := strings.ToLower(col.Schema) + "." + strings.ToLower(col.Table)
			if base, ok := m.bySchemaName[key]; ok {
				add(base)
			}
		case col.Table != "":
			// Try alias first (always unambiguous).
			if base, ok := m.byAlias[strings.ToLower(col.Table)]; ok {
				add(base)
				continue
			}
			// Bare-name lookup. Single match → use; multiple → skip
			// (ambiguous user reference under joined same-named
			// tables; without schema info we can't disambiguate).
			bases := m.byBareName[strings.ToLower(col.Table)]
			if len(bases) == 1 {
				add(bases[0])
			}
		default:
			// Unqualified SET column.
			// Single-target shortcut (Codex-fix-1b: count DISTINCT
			// base tables, not lookup-map entries).
			if len(m.distinctBases) == 1 {
				add(m.distinctBases[0])
				continue
			}
			// Multi-target: schema-aware column resolution
			// (Codex-fix-1e, refined by Codex-fix-1h to mirror
			// transformer semantics — any-match attributes; zero-
			// match or no-metadata falls back to all distinctBases).
			for _, t := range omniResolveUnqualifiedSETColumn(col.Column, m.distinctBases, dbMetadata) {
				add(t)
			}
		}
	}
	return result
}

// omniResolveUnqualifiedSETColumn resolves an unqualified SET column
// name to its owning base table(s). Mirrors the transformer's
// resolveUnqualifiedColumns semantics at
// parser/tidb/backup.go:539-576 (cumulative #30 Codex-fix-1h):
//
//   - No metadata available → return ALL distinctBases (fallback).
//   - Column found in N≥1 distinct bases → return THOSE N bases.
//     Single-match is the common case (Codex-fix-1e — typical
//     joins-for-filtering pattern); multi-match means the column
//     is ambiguous (MySQL itself errors at execution time, but
//     the advisor signals the broader risk surface).
//   - Column not found in ANY base → return ALL distinctBases
//     (fallback — column likely added by an earlier statement
//     or schema lag; the transformer treats this conservatively
//     and the advisor must too to avoid approving inputs the
//     transformer rejects).
//
// Earlier Codex-fix-1e returned nil on ambiguous/zero matches —
// that diverged from transformer behavior and let multi-table
// count-cap violations slip through (Codex P1 #10). Refined here
// to mirror the transformer; the advisor stays as the prediction
// layer for the transformer's runtime constraint.
//
// Schema-match policy: a base whose database doesn't match
// dbMetadata.Name is excluded from the catalog walk — we have
// no catalog info for it. Cross-database UPDATEs with unqualified
// SET fall through to the zero-match path → all distinctBases
// fallback. Case-insensitive matching on table/column names per
// MySQL convention.
func omniResolveUnqualifiedSETColumn(colName string, distinctBases []priorBackupTable, dbMetadata *storepb.DatabaseSchemaMetadata) []priorBackupTable {
	if colName == "" {
		return distinctBases
	}
	if dbMetadata == nil {
		return distinctBases
	}
	dbName := dbMetadata.GetName()
	var matches []priorBackupTable
	for _, base := range distinctBases {
		// Cross-DB bases: no catalog info; skip the catalog walk
		// for these. They still participate in the fallback path
		// when overall match count is zero.
		if base.database != "" && dbName != "" && !strings.EqualFold(base.database, dbName) {
			continue
		}
		for _, schema := range dbMetadata.Schemas {
			for _, table := range schema.Tables {
				if !strings.EqualFold(table.Name, base.table) {
					continue
				}
				for _, c := range table.Columns {
					if strings.EqualFold(c.Name, colName) {
						matches = append(matches, base)
						break
					}
				}
			}
		}
	}
	if len(matches) > 0 {
		return matches
	}
	return distinctBases
}

// (omniIsDDLStmt removed in cumulative #30 Codex-fix-1f. DDL
// detection uses pingcap's DDLNode interface directly via the
// getTiDBNodes path — handles Tier-4-deferred grammar that omni
// rejects at parse time. Full lesson + 9-sub-fix algorithm-
// corrections lineage lives in plan-doc cumulative #30.)
