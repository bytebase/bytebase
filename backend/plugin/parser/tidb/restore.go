package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	maxCommentLength = 1000

	// errMsgFailedToGetSourceDB is the wrap-error format string used at
	// every common.GetInstanceDatabaseID(SourceTable.Database) call site.
	// Centralized to avoid string drift across the three error-wrap sites
	// (GenerateRestoreSQL / doGenerate / extractStatement).
	errMsgFailedToGetSourceDB = "failed to get source database ID for %s"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_TIDB, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	// Nil guard: backupItem and its SourceTable/TargetTable sub-fields are
	// dereferenced unconditionally below. Pre-Bug-9 the nil-backupItem check
	// lived in extractStatement (called first); the Bug 9 refactor moved
	// metadata fetching ahead of extractStatement, bypassing that guard.
	// Per Codex P2 catch on PR #20345 — restoring the nil-backupItem check
	// AND adding sub-field checks since each is independently dereferenced.
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}
	if backupItem.SourceTable == nil {
		return "", errors.Errorf("backup item source table is nil")
	}
	if backupItem.TargetTable == nil {
		return "", errors.Errorf("backup item target table is nil")
	}

	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, errMsgFailedToGetSourceDB, backupItem.SourceTable.Database)
	}

	// Fetch the target table's regular column set once. Used by
	// updateMutatesTable to resolve unqualified SET columns against the
	// target's actual schema — without it, an unqualified SET on a
	// non-target-table column would mis-classify the UPDATE as mutating
	// the target and produce invalid `... ON DUPLICATE KEY UPDATE ;`
	// downstream. Per Codex P1 catch on PR #20345.
	targetCols, err := getNormalColumnsLower(ctx, rCtx, sourceDatabase, backupItem.SourceTable.Table)
	if err != nil {
		return "", errors.Wrap(err, "failed to get target table columns")
	}

	originalSQL, err := extractStatement(statement, backupItem, sourceDatabase, targetCols)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	// Find ALL DML nodes referencing the target table — backup.go's single-
	// table path bundles >maxMixedDMLCount same-table DMLs into one backup
	// item, and rollback must cover every column touched across the bundle
	// (not just the first stmt's columns). Per Codex P1 catch on PR #20345.
	matchingNodes, err := findMatchingDMLs(originalSQL, sourceDatabase, backupItem.SourceTable.Table, targetCols)
	if err != nil {
		return "", err
	}
	if len(matchingNodes) == 0 {
		return "", errors.Errorf("no DML statement found in extracted SQL")
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}
	return doGenerate(ctx, rCtx, sqlForComment, matchingNodes, backupItem)
}

// getNormalColumnsLower returns a lowercased set of regular (non-
// generated) column names for the given table. Used by
// updateMutatesTable to determine whether an unqualified SET column
// belongs to the target table — without this, multi-table UPDATEs
// where unqualified SETs reference columns of the joined-but-not-
// target table would be mis-classified as mutating the target.
func getNormalColumnsLower(ctx context.Context, rCtx base.RestoreContext, database, table string) (map[string]bool, error) {
	if rCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.Errorf("GetDatabaseMetadataFunc is nil")
	}
	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for %s", database)
	}
	if metadata == nil {
		return nil, errors.Errorf("database metadata is nil for %s", database)
	}
	schema := metadata.GetSchemaMetadata("")
	if schema == nil {
		return nil, errors.Errorf("schema is nil for %s", database)
	}
	tableMetadata := schema.GetTable(table)
	if tableMetadata == nil {
		return nil, errors.Errorf("table metadata is nil for %s.%s", database, table)
	}
	result := make(map[string]bool)
	for _, col := range tableMetadata.GetProto().Columns {
		if col == nil || col.Generation != nil {
			continue
		}
		result[strings.ToLower(col.Name)] = true
	}
	return result, nil
}

// findMatchingDMLs returns ALL UPDATE/DELETE nodes in the parsed statement
// list that reference the target table. The single-DML form (return-on-
// first-match) is incorrect for backup.go's single-table-bundling path,
// where one backup item spans multiple same-table DMLs touching different
// columns; rolling back only the first stmt's columns leaves later
// columns mutated. See Codex P1 catch on PR #20345.
func findMatchingDMLs(statement, database, table string, targetCols map[string]bool) ([]ast.Node, error) {
	stmtList, err := ParseTiDBOmni(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}
	var result []ast.Node
	for _, item := range stmtList.Items {
		switch item.(type) {
		case *ast.UpdateStmt, *ast.DeleteStmt:
			if containsTable(item, database, table, targetCols) {
				result = append(result, item)
			}
		default:
		}
	}
	return result, nil
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, nodes []ast.Node, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, errMsgFailedToGetSourceDB, backupItem.SourceTable.Database)
	}
	_, targetDatabase, err := common.GetInstanceDatabaseID(backupItem.TargetTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get target database ID for %s", backupItem.TargetTable.Database)
	}
	generatedColumns, normalColumns, err := classifyColumns(ctx, rCtx.GetDatabaseMetadataFunc, rCtx.ListDatabaseNamesFunc, rCtx.IsCaseSensitive, rCtx.InstanceID, &TableReference{
		Database: sourceDatabase,
		Table:    backupItem.SourceTable.Table,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to classify columns for %s.%s", backupItem.SourceTable.Database, backupItem.SourceTable.Table)
	}

	// Partition nodes by type. If any UPDATE exists in the bundle, ODKU
	// is required — pure INSERT SELECT would FAIL on the surviving UPDATE-
	// modified rows due to duplicate key. ODKU restores the listed columns
	// for those rows AND inserts the deleted rows (no conflict). If the
	// bundle is pure-DELETE, simple INSERT SELECT suffices (no surviving
	// rows means no conflicts).
	var updateStmts []*ast.UpdateStmt
	for _, n := range nodes {
		switch v := n.(type) {
		case *ast.UpdateStmt:
			updateStmts = append(updateStmts, v)
		case *ast.DeleteStmt:
			// DELETE doesn't contribute columns; tracked only by partition.
		default:
			return "", errors.Errorf("unexpected statement type: %T", n)
		}
	}

	var result string
	if len(updateStmts) > 0 {
		r, err := generateUpdateRestore(ctx, rCtx, updateStmts, sourceDatabase, backupItem.SourceTable.Table, targetDatabase, backupItem.TargetTable.Table, generatedColumns, normalColumns)
		if err != nil {
			return "", err
		}
		result = r
	} else {
		result = generateDeleteRestore(sourceDatabase, backupItem.SourceTable.Table, targetDatabase, backupItem.TargetTable.Table, generatedColumns, normalColumns)
	}

	return fmt.Sprintf("/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, result), nil
}

func generateDeleteRestore(originalDatabase, originalTable, backupDatabase, backupTable string, generatedColumns, normalColumns []string) string {
	if len(generatedColumns) == 0 {
		return fmt.Sprintf("INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s`;", originalDatabase, originalTable, backupDatabase, backupTable)
	}
	var quotedColumns []string
	for _, column := range normalColumns {
		quotedColumns = append(quotedColumns, fmt.Sprintf("`%s`", column))
	}
	quotedColumnList := strings.Join(quotedColumns, ", ")
	return fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) SELECT %s FROM `%s`.`%s`;", originalDatabase, originalTable, quotedColumnList, quotedColumnList, backupDatabase, backupTable)
}

func generateUpdateRestore(ctx context.Context, rCtx base.RestoreContext, stmts []*ast.UpdateStmt, originalDatabase, originalTable, backupDatabase, backupTable string, generatedColumns, normalColumns []string) (string, error) {
	// Union update columns across ALL stmts in the bundle. backup.go's
	// single-table path bundles >maxMixedDMLCount same-table DMLs into one
	// backup_item; restore must rollback every column touched across the
	// whole bundle, not just the first stmt's columns. Per Codex P1 catch
	// on PR #20345.
	//
	// Iteration order is preserved (first-seen-first) for deterministic
	// ODKU clause output; case-insensitive dedup matches the
	// hasDisjointUniqueKey lower-case map convention.
	seen := make(map[string]bool)
	var updateColumns []string
	for _, stmt := range stmts {
		singleTables := extractSingleTablesFromTableExprs(originalDatabase, stmt.Tables)
		for _, col := range extractUpdateColumns(stmt.SetList, originalDatabase, originalTable, singleTables, normalColumns) {
			key := strings.ToLower(col)
			if !seen[key] {
				seen[key] = true
				updateColumns = append(updateColumns, col)
			}
		}
	}

	has, err := hasDisjointUniqueKey(ctx, rCtx, originalDatabase, originalTable, updateColumns)
	if err != nil {
		return "", err
	}
	if !has {
		return "", errors.Errorf("no disjoint unique key found for %s.%s", originalDatabase, originalTable)
	}

	var buf strings.Builder
	if len(generatedColumns) == 0 {
		if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s` ON DUPLICATE KEY UPDATE ", originalDatabase, originalTable, backupDatabase, backupTable); err != nil {
			return "", err
		}
	} else {
		var quotedColumns []string
		for _, column := range normalColumns {
			quotedColumns = append(quotedColumns, fmt.Sprintf("`%s`", column))
		}
		quotedColumnList := strings.Join(quotedColumns, ", ")
		if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` (%s) SELECT %s FROM `%s`.`%s` ON DUPLICATE KEY UPDATE ", originalDatabase, originalTable, quotedColumnList, quotedColumnList, backupDatabase, backupTable); err != nil {
			return "", err
		}
	}

	for i, field := range updateColumns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprintf(&buf, "`%s` = VALUES(`%s`)", field, field); err != nil {
			return "", err
		}
	}
	if _, err := buf.WriteString(";"); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// extractSingleTablesFromTableExprs walks omni TableExpr nodes and returns
// a map of alias-or-name to TableReference for simple table references.
func extractSingleTablesFromTableExprs(databaseName string, exprs []ast.TableExpr) map[string]*TableReference {
	result := make(map[string]*TableReference)
	for _, expr := range exprs {
		collectTableRefs(databaseName, expr, result)
	}
	return result
}

func collectTableRefs(databaseName string, expr ast.TableExpr, result map[string]*TableReference) {
	switch n := expr.(type) {
	case *ast.TableRef:
		database := n.Schema
		if database == "" {
			database = databaseName
		}
		table := &TableReference{
			Database: database,
			Table:    n.Name,
			Alias:    n.Alias,
		}
		// Map keys are normalized to lowercase to match TiDB's
		// case-insensitive identifier semantics — a SET clause that
		// references the alias with different case (e.g., `T1` declared,
		// `t1` referenced in SET) must still resolve. Per Codex P1
		// follow-on catch on PR #20345.
		var key string
		if n.Alias != "" {
			key = strings.ToLower(n.Alias)
		} else {
			key = strings.ToLower(n.Name)
		}
		result[key] = table
	case *ast.JoinClause:
		collectTableRefs(databaseName, n.Left, result)
		collectTableRefs(databaseName, n.Right, result)
	default:
	}
}

// extractUpdateColumns extracts column names from SET assignments that
// belong to the original table.
//
// For qualified SET columns (`SET t1.a = ...`), the qualifier is looked
// up in singleTables (the alias-or-name → TableReference map for this
// stmt) — if the qualifier resolves to a TableReference whose .Table
// equals originalTable, the column is included. This is deterministic
// even for self-join UPDATEs where the same physical table appears
// under multiple aliases (e.g., `UPDATE test t1 JOIN test t2 SET t1.a
// = ...`); each SET clause's qualifier independently resolves to the
// correct alias entry. Per Codex P1 catch on PR #20345.
//
// Pre-fix this function took a single matchedTable picked by the caller
// via map iteration over singleTables. Map iteration is randomized in
// Go, so for self-joins the wrong alias could be picked, leading to
// empty result and invalid `... ON DUPLICATE KEY UPDATE ;` rollback SQL.
func extractUpdateColumns(setList []*ast.Assignment, database, originalTable string, singleTables map[string]*TableReference, normalColumns []string) []string {
	var result []string
	for _, assignment := range setList {
		col := assignment.Column
		if col == nil {
			continue
		}

		if col.Schema != "" && !strings.EqualFold(col.Schema, database) {
			continue
		}

		if col.Table == "" {
			// Unqualified column: check if it belongs to the normalColumns set.
			for _, c := range normalColumns {
				if strings.EqualFold(c, col.Column) {
					result = append(result, col.Column)
					break
				}
			}
			continue
		}

		// Qualified column: resolve the qualifier through the alias map
		// (handles self-joins deterministically). Lookup key is
		// lowercased to match the case-insensitive insert convention
		// in collectTableRefs. BOTH Database AND Table must match —
		// without the Database check, cross-database joins with
		// homonymous tables (e.g. `UPDATE db1.test t1 JOIN db2.test t2
		// SET t2.a = ...`) would incorrectly include the joined-DB
		// SET column in the target-DB's rollback. Per Codex P1 catch
		// on PR #20345.
		if entry, ok := singleTables[strings.ToLower(col.Table)]; ok && strings.EqualFold(entry.Database, database) && strings.EqualFold(entry.Table, originalTable) {
			result = append(result, col.Column)
			continue
		}
		// Fallback: qualifier IS the bare table name (no alias used).
		// col.Schema check above already filtered explicit-schema
		// mismatches.
		if strings.EqualFold(col.Table, originalTable) {
			result = append(result, col.Column)
		}
	}
	return result
}

func hasDisjointUniqueKey(ctx context.Context, rCtx base.RestoreContext, originalDatabase, originalTable string, updateColumns []string) (bool, error) {
	columnMap := make(map[string]bool)
	for _, column := range updateColumns {
		columnMap[strings.ToLower(column)] = true
	}

	if rCtx.GetDatabaseMetadataFunc == nil {
		return false, errors.Errorf("GetDatabaseMetadataFunc is nil")
	}

	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, originalDatabase)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get database metadata for %s", originalDatabase)
	}
	if metadata == nil {
		return false, errors.Errorf("database metadata is nil for %s", originalDatabase)
	}

	schema := metadata.GetSchemaMetadata("")
	if schema == nil {
		return false, errors.Errorf("schema is nil for %s", originalDatabase)
	}

	tableMetadata := schema.GetTable(originalTable)
	if tableMetadata == nil {
		return false, errors.Errorf("table metadata is nil for %s.%s", originalDatabase, originalTable)
	}

	tableProto := tableMetadata.GetProto()

	// Build a fast lookup of regular (non-generated) column names. Used
	// to filter out unique keys whose Expressions reference generated
	// columns or non-column expressions (functional indexes) — those are
	// unsafe for ODKU rollback because string-comparison disjoint can't
	// tell whether the SET clause indirectly affects them. Per peer
	// review on PR #20345 (Finding 4).
	regularColumns := make(map[string]bool)
	for _, col := range tableProto.Columns {
		if col == nil || col.Generation != nil {
			continue
		}
		regularColumns[strings.ToLower(col.Name)] = true
	}

	for _, index := range tableProto.Indexes {
		if !index.Primary && !index.Unique {
			continue
		}
		// Skip UKs whose Expressions reference anything other than
		// regular (non-generated) columns. A UK on a generated column
		// (e.g., `c_generated = a + b`) appears disjoint from SET cols
		// {a, b} via string comparison — but updating a or b changes
		// c_generated's value, so the UK is NOT safe for ODKU matching.
		// Same problem for functional indexes (Expressions = `(LOWER(email))`).
		// Conservative: treat any UK with non-regular-column expressions
		// as overlapping (skip). Never returns false-positive disjoint.
		//
		// ALSO skip UKs with empty Expressions — disjoint([]) returns
		// vacuously true, which would falsely mark the UK as safe.
		// TiDB metadata produces empty Expressions for some expression-
		// based index parts (per backend/plugin/schema/tidb/
		// get_database_metadata.go:getIndexColumnsInfo, parts without
		// key.Column aren't added). Per Codex P1 catch on PR #20345.
		if len(index.Expressions) == 0 {
			continue
		}
		safe := true
		for _, expr := range index.Expressions {
			if !regularColumns[strings.ToLower(expr)] {
				safe = false
				break
			}
		}
		if !safe {
			continue
		}
		if disjoint(index.Expressions, columnMap) {
			return true, nil
		}
	}

	return false, nil
}

func disjoint(a []string, b map[string]bool) bool {
	for _, item := range a {
		if _, ok := b[strings.ToLower(item)]; ok {
			return false
		}
	}
	return true
}

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item, sourceDatabase string, targetCols map[string]bool) (string, error) {
	// Nil-backupItem guard now lives in GenerateRestoreSQL (the only
	// caller); extractStatement is unexported and assumes non-nil input.

	list, err := SplitSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to split sql")
	}

	start := 0
	end := len(list) - 1
	for i, item := range list {
		if equalOrLess(item.Start, backupItem.StartPosition) {
			start = i
		}
	}

	for i := len(list) - 1; i >= 0; i-- {
		// EndPosition is the EXCLUSIVE end of the range (per
		// base/statement.go:16-18, "points to the position AFTER the last
		// character of the statement"). A stmt whose Start is AT or AFTER
		// EndPosition belongs to the NEXT backup item, not this one — so
		// it must be EXCLUDED from this slice. Pre-fix used `end = i`
		// which included the boundary stmt; in mixed-DML mode (each
		// backup item maps to one stmt), that bled the next backup
		// item's stmt into this one's rollback. Per Codex P1 catch on
		// PR #20345.
		if equalOrGreater(list[i].Start, backupItem.EndPosition) {
			end = i - 1
		}
	}

	var result []string
	for i := start; i <= end; i++ {
		stmtList, err := ParseTiDBOmni(list[i].Text)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse sql")
		}
		containsSourceTable := false
		for _, node := range stmtList.Items {
			if containsTable(node, sourceDatabase, backupItem.SourceTable.Table, targetCols) {
				containsSourceTable = true
				break
			}
		}
		if containsSourceTable {
			result = append(result, list[i].Text)
		}
	}
	return strings.Join(result, ""), nil
}

// containsTable checks whether the given AST node MUTATES the specified
// table (not merely references it via JOIN/USING).
//
// For UPDATE: the table must appear as the qualifier of at least one SET
// assignment — being only joined for filtering is not enough.
//
// For DELETE: n.Tables holds the explicit delete-targets (mutation set);
// n.Using holds JOIN-only refs (filter set). Only n.Tables counts as
// mutation. Per Codex P1 follow-on catch on PR #20345 — pre-fix
// matching n.Using too caused a backup item targeting a USING-only
// table to generate rollback SQL that re-inserted rows that were never
// deleted (reintroducing stale data).
func containsTable(node ast.Node, database, table string, targetCols map[string]bool) bool {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		return updateMutatesTable(n, database, table, targetCols)
	case *ast.DeleteStmt:
		for _, expr := range n.Tables {
			if tableExprReferences(expr, database, table) {
				return true
			}
		}
		// n.Using is JOIN-only (filter set), not mutation — do NOT match.
	default:
	}
	return false
}

// updateMutatesTable reports whether the UPDATE's SET clauses actually
// mutate (a column of) the specified table. JOIN-only references that
// do not appear as a SET-clause qualifier do NOT count as mutation.
//
// Resolution strategy mirrors extractUpdateColumns: build the alias map
// (lowercased per Bug 5), then for each SET assignment resolve the
// qualifier through the map. Unqualified columns are conservatively
// counted as mutation if the target table is in scope (multi-table
// UPDATE with unqualified col is ambiguous; safer to over-include than
// miss).
func updateMutatesTable(stmt *ast.UpdateStmt, database, table string, targetCols map[string]bool) bool {
	// Precondition: target must be in stmt.Tables at all (else there's
	// nothing to discuss).
	targetInScope := false
	for _, expr := range stmt.Tables {
		if tableExprReferences(expr, database, table) {
			targetInScope = true
			break
		}
	}
	if !targetInScope {
		return false
	}

	singleTables := extractSingleTablesFromTableExprs(database, stmt.Tables)
	for _, assignment := range stmt.SetList {
		col := assignment.Column
		if col == nil {
			continue
		}
		if col.Schema != "" && !strings.EqualFold(col.Schema, database) {
			continue
		}
		if col.Table == "" {
			// Unqualified column. Resolve against the target table's
			// actual columns — only count as mutation if the column
			// exists on the target. Pre-fix this branch returned true
			// unconditionally (over-counted); for `UPDATE test JOIN t1
			// ON ... SET name = 1` where `name` is on t1 but not test,
			// the over-count produced empty `... ON DUPLICATE KEY
			// UPDATE ;`. Per Codex P1 catch on PR #20345.
			if targetCols[strings.ToLower(col.Column)] {
				return true
			}
			continue
		}
		// Qualified — resolve qualifier through the alias map. Both
		// Database AND Table must match: for cross-database joins with
		// homonymous tables (e.g. `UPDATE db1.test t1 JOIN db2.test t2
		// SET t2.a = ...`), the alias t2 resolves to db2.test; without
		// the Database check, a backup item targeting db1.test would
		// incorrectly match db2.test's SET assignments. Per Codex P1
		// catch on PR #20345.
		if entry, ok := singleTables[strings.ToLower(col.Table)]; ok && strings.EqualFold(entry.Database, database) && strings.EqualFold(entry.Table, table) {
			return true
		}
		// Fallback: qualifier IS the bare table name (no alias used).
		// The col.Schema check at the top of the loop already filtered
		// out assignments whose explicit schema doesn't match `database`,
		// so reaching here implies col.Schema is empty or matches.
		if strings.EqualFold(col.Table, table) {
			return true
		}
	}
	return false
}

func tableExprReferences(expr ast.TableExpr, database, table string) bool {
	switch n := expr.(type) {
	case *ast.TableRef:
		db := n.Schema
		if db == "" {
			db = database
		}
		// Both comparisons are case-insensitive — TiDB/MySQL identifier
		// comparisons are typically case-insensitive in practice, and the
		// table-name comparison below already uses EqualFold; using == on
		// the database side would inconsistently miss schema-qualified
		// references like `DB.test` when backupItem stores `db`.
		return strings.EqualFold(db, database) && strings.EqualFold(n.Name, table)
	case *ast.JoinClause:
		return tableExprReferences(n.Left, database, table) || tableExprReferences(n.Right, database, table)
	}
	return false
}

func equalOrLess(a, b *storepb.Position) bool {
	if a.Line < b.Line {
		return true
	}
	if a.Line == b.Line && a.Column <= b.Column {
		return true
	}
	return false
}

func equalOrGreater(a, b *storepb.Position) bool {
	if a.Line > b.Line {
		return true
	}
	if a.Line == b.Line && a.Column >= b.Column {
		return true
	}
	return false
}
