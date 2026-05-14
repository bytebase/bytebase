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
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_TIDB, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	// Find ALL DML nodes referencing the target table — backup.go's single-
	// table path bundles >maxMixedDMLCount same-table DMLs into one backup
	// item, and rollback must cover every column touched across the bundle
	// (not just the first stmt's columns). Per Codex P1 catch on PR #20345.
	matchingNodes, err := findMatchingDMLs(originalSQL, sourceDatabase, backupItem.SourceTable.Table)
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

// findMatchingDMLs returns ALL UPDATE/DELETE nodes in the parsed statement
// list that reference the target table. The single-DML form (return-on-
// first-match) is incorrect for backup.go's single-table-bundling path,
// where one backup item spans multiple same-table DMLs touching different
// columns; rolling back only the first stmt's columns leaves later
// columns mutated. See Codex P1 catch on PR #20345.
func findMatchingDMLs(statement, database, table string) ([]ast.Node, error) {
	stmtList, err := ParseTiDBOmni(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}
	if stmtList == nil {
		return nil, nil
	}
	var result []ast.Node
	for _, item := range stmtList.Items {
		switch item.(type) {
		case *ast.UpdateStmt, *ast.DeleteStmt:
			if containsTable(item, database, table) {
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
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
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
		if n.Alias != "" {
			result[n.Alias] = table
		} else {
			result[n.Name] = table
		}
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
		// (handles self-joins deterministically).
		if entry, ok := singleTables[col.Table]; ok && strings.EqualFold(entry.Table, originalTable) {
			result = append(result, col.Column)
			continue
		}
		// Fallback: qualifier IS the bare table name (no alias used).
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

	for _, index := range tableMetadata.GetProto().Indexes {
		if !index.Primary && !index.Unique {
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

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

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

	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	var result []string
	for i := start; i <= end; i++ {
		stmtList, err := ParseTiDBOmni(list[i].Text)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse sql")
		}
		containsSourceTable := false
		for _, node := range stmtList.Items {
			if containsTable(node, sourceDatabase, backupItem.SourceTable.Table) {
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

// containsTable checks whether the given AST node references the specified table.
func containsTable(node ast.Node, database, table string) bool {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		for _, expr := range n.Tables {
			if tableExprReferences(expr, database, table) {
				return true
			}
		}
	case *ast.DeleteStmt:
		for _, expr := range n.Tables {
			if tableExprReferences(expr, database, table) {
				return true
			}
		}
		for _, expr := range n.Using {
			if tableExprReferences(expr, database, table) {
				return true
			}
		}
	default:
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
