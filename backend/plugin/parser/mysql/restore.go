package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MYSQL, GenerateRestoreSQL)
	base.RegisterGenerateRestoreSQL(storepb.Engine_MARIADB, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	// Parse the filtered SQL and find the first DML node.
	matchingNode, err := findFirstDML(originalSQL)
	if err != nil {
		return "", err
	}
	if matchingNode == nil {
		return "", errors.Errorf("no DML statement found in extracted SQL")
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}
	return doGenerate(ctx, rCtx, sqlForComment, matchingNode, backupItem)
}

func findFirstDML(statement string) (ast.Node, error) {
	stmtList, err := ParseMySQLOmni(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}
	if stmtList == nil {
		return nil, nil
	}
	for _, item := range stmtList.Items {
		switch item.(type) {
		case *ast.UpdateStmt, *ast.DeleteStmt:
			return item, nil
		default:
		}
	}
	return nil, nil
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, node ast.Node, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
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

	var result string
	switch n := node.(type) {
	case *ast.DeleteStmt:
		result = generateDeleteRestore(sourceDatabase, backupItem.SourceTable.Table, targetDatabase, backupItem.TargetTable.Table, generatedColumns, normalColumns)
	case *ast.UpdateStmt:
		r, err := generateUpdateRestore(ctx, rCtx, n, sourceDatabase, backupItem.SourceTable.Table, targetDatabase, backupItem.TargetTable.Table, generatedColumns, normalColumns)
		if err != nil {
			return "", err
		}
		result = r
	default:
		return "", errors.Errorf("unexpected statement type: %T", node)
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

func generateUpdateRestore(ctx context.Context, rCtx base.RestoreContext, stmt *ast.UpdateStmt, originalDatabase, originalTable, backupDatabase, backupTable string, generatedColumns, normalColumns []string) (string, error) {
	// Extract single tables from the UPDATE table references.
	singleTables := extractSingleTablesFromTableExprs(originalDatabase, stmt.Tables)

	// Find the table matching the original table.
	var matchedTable *TableReference
	for _, table := range singleTables {
		if strings.EqualFold(table.Table, originalTable) {
			matchedTable = table
			break
		}
	}

	// Extract update column names that belong to the original table.
	updateColumns := extractUpdateColumns(stmt.SetList, originalDatabase, originalTable, matchedTable, normalColumns)

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

// extractUpdateColumns extracts column names from SET assignments that belong
// to the original table.
func extractUpdateColumns(setList []*ast.Assignment, database, _ string, matchedTable *TableReference, normalColumns []string) []string {
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

		if matchedTable != nil && (col.Table == matchedTable.Alias || strings.EqualFold(col.Table, matchedTable.Table)) {
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
		if equalOrGreater(list[i].Start, backupItem.EndPosition) {
			end = i
		}
	}

	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	var result []string
	for i := start; i <= end; i++ {
		stmtList, err := ParseMySQLOmni(list[i].Text)
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
		return db == database && strings.EqualFold(n.Name, table)
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
