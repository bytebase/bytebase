package tidb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	maxTableNameLength = 64
	maxMixedDMLCount   = 5
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_TIDB, TransformDMLToSelect)
}

type TableReference struct {
	Database string
	Table    string
	Alias    string
}

type statementInfo struct {
	offset        int
	statement     string
	table         *TableReference
	node          ast.Node
	fullSQL       string
	startPosition *storepb.Position
	endPosition   *storepb.Position
}

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	var dbMetadata *model.DatabaseMetadata
	if tCtx.GetDatabaseMetadataFunc != nil {
		_, metadata, err := tCtx.GetDatabaseMetadataFunc(ctx, tCtx.InstanceID, sourceDatabase)
		if err == nil {
			dbMetadata = metadata
		}
	}

	statementInfoList, err := prepareTransformation(sourceDatabase, statement, dbMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(ctx, tCtx, statementInfoList, targetDatabase, tablePrefix)
}

func prepareTransformation(databaseName, statement string, dbMetadata *model.DatabaseMetadata) ([]statementInfo, error) {
	list, err := SplitSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split sql")
	}

	var result []statementInfo
	for i, item := range list {
		if len(item.Text) == 0 || item.Empty {
			continue
		}

		parsed, err := ParseTiDBOmni(item.Text)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse sql")
		}
		if parsed == nil || len(parsed.Items) == 0 {
			continue
		}

		for _, node := range parsed.Items {
			tables, err := extractTables(databaseName, node, item.Text, dbMetadata)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract tables")
			}
			for _, table := range tables {
				// A BackupStatement carries no source database, and the executor
				// records the task database for the backed-up table. Reject
				// cross-database mutations (DELETE or UPDATE) so a rollback can't
				// be written to the wrong database.
				if table.table.Database != databaseName {
					return nil, errors.Errorf("prior backup does not support cross-database mutations: %s.%s (task database %q)", table.table.Database, table.table.Table, databaseName)
				}
				table.offset = i
				table.startPosition = item.Start
				table.endPosition = item.End
				result = append(result, table)
			}
		}
	}

	return result, nil
}

// extractTables extracts the affected table references from a single DML node.
func extractTables(databaseName string, node ast.Node, fullSQL string, dbMetadata *model.DatabaseMetadata) ([]statementInfo, error) {
	switch n := node.(type) {
	case *ast.DeleteStmt:
		return extractTablesFromDelete(databaseName, n, fullSQL)
	case *ast.UpdateStmt:
		return extractTablesFromUpdate(databaseName, n, fullSQL, dbMetadata)
	case *ast.BatchStmt:
		// TiDB BATCH (non-transactional DML) is not supported by prior backup.
		// Reject it explicitly rather than returning an empty list, which the
		// task executor would treat as a successful no-op and then run the
		// mutation with no backup.
		return nil, errors.New("prior backup does not support TiDB BATCH (non-transactional DML) statements")
	default:
		return nil, nil
	}
}

func extractTablesFromDelete(databaseName string, n *ast.DeleteStmt, fullSQL string) ([]statementInfo, error) {
	cteNames := collectCTENames(fullSQL, n.Loc)
	stmtText := extractStatementText(fullSQL, n.Loc)

	singleTables := collectSingleTables(databaseName, n.Tables)

	if len(n.Using) == 0 {
		// Single-table DELETE: DELETE FROM t WHERE ...
		if len(singleTables) == 0 {
			return nil, nil
		}
		first := singleTables[firstKey(singleTables)]
		return []statementInfo{{
			statement: stmtText,
			table:     first,
			node:      n,
			fullSQL:   fullSQL,
		}}, nil
	}

	// Multi-table DELETE: DELETE t1, t2 FROM t1 JOIN t2 ... WHERE ...
	// Tables = targets to delete from; Using = the referenced table list.
	refTables := collectSingleTables(databaseName, n.Using)

	var result []statementInfo
	for _, target := range singleTables {
		name := target.Table
		if target.Alias != "" {
			name = target.Alias
		}
		if cteNames[name] {
			continue
		}
		ref, ok := refTables[name]
		if !ok {
			return nil, errors.Errorf("cannot extract reference table: no matched table %q in referenced table list", name)
		}
		result = append(result, statementInfo{
			statement: stmtText,
			table:     ref,
			node:      n,
			fullSQL:   fullSQL,
		})
	}
	return result, nil
}

func extractTablesFromUpdate(databaseName string, n *ast.UpdateStmt, fullSQL string, dbMetadata *model.DatabaseMetadata) ([]statementInfo, error) {
	cteNames := collectCTENames(fullSQL, n.Loc)
	stmtText := extractStatementText(fullSQL, n.Loc)

	singleTables := collectSingleTables(databaseName, n.Tables)

	// Determine which tables the SET clause writes, via column table prefixes.
	updatedTables := make(map[string]bool)
	var unqualifiedColumns []string
	for _, assign := range n.SetList {
		if assign == nil || assign.Column == nil {
			continue
		}
		table := assign.Column.Table
		if cteNames[table] {
			continue
		}
		updatedTables[table] = true
		if table == "" {
			unqualifiedColumns = append(unqualifiedColumns, assign.Column.Column)
		}
	}

	// Resolve unqualified SET columns to their owning table(s) via metadata.
	// Exclude CTE refs from the candidates: a CTE can't be a mutation target,
	// and resolveUnqualifiedColumns' metadata-unavailable/unresolved fallback
	// would otherwise mark every candidate (CTEs included).
	if updatedTables[""] {
		delete(updatedTables, "")
		candidates := make(map[string]*TableReference, len(singleTables))
		for key, ref := range singleTables {
			if cteNames[key] || cteNames[ref.Table] {
				continue
			}
			candidates[key] = ref
		}
		for t := range resolveUnqualifiedColumns(unqualifiedColumns, candidates, dbMetadata) {
			updatedTables[t] = true
		}
	}

	// Single-table UPDATE: no explicit qualification needed.
	if len(updatedTables) == 0 && len(singleTables) == 1 {
		for _, ref := range singleTables {
			return []statementInfo{{
				statement: stmtText,
				table:     ref,
				node:      n,
				fullSQL:   fullSQL,
			}}, nil
		}
	}

	var result []statementInfo
	for table := range updatedTables {
		ref, ok := singleTables[table]
		if !ok {
			return nil, errors.Errorf("cannot extract reference table: no matched updated table %q in referenced table list", table)
		}
		result = append(result, statementInfo{
			statement: stmtText,
			table:     ref,
			node:      n,
			fullSQL:   fullSQL,
		})
	}
	return result, nil
}

// collectSingleTables walks TableExpr slices and collects TableReference entries
// keyed by alias or table name. A table's database is its explicit schema, or
// the statement's database when unqualified.
func collectSingleTables(databaseName string, exprs []ast.TableExpr) map[string]*TableReference {
	result := make(map[string]*TableReference)
	for _, expr := range exprs {
		collectSingleTablesFromExpr(databaseName, expr, result)
	}
	return result
}

func collectSingleTablesFromExpr(databaseName string, expr ast.TableExpr, out map[string]*TableReference) {
	switch e := expr.(type) {
	case *ast.TableRef:
		db := e.Schema
		if db == "" {
			db = databaseName
		}
		ref := &TableReference{Database: db, Table: e.Name, Alias: e.Alias}
		key := e.Name
		if e.Alias != "" {
			key = e.Alias
		}
		out[key] = ref
	case *ast.JoinClause:
		if e.Left != nil {
			collectSingleTablesFromExpr(databaseName, e.Left, out)
		}
		if e.Right != nil {
			collectSingleTablesFromExpr(databaseName, e.Right, out)
		}
	default:
	}
}

// resolveUnqualifiedColumns resolves unqualified SET column names to their
// owning table(s) using metadata. Falls back to all referenced tables when
// metadata is unavailable or a column cannot be located.
func resolveUnqualifiedColumns(columns []string, singleTables map[string]*TableReference, dbMetadata *model.DatabaseMetadata) map[string]bool {
	result := make(map[string]bool)

	var schema *model.SchemaMetadata
	if dbMetadata != nil {
		schema = dbMetadata.GetSchemaMetadata("")
	}
	if schema == nil {
		for t := range singleTables {
			result[t] = true
		}
		return result
	}

	for _, col := range columns {
		resolved := false
		for aliasOrName, ref := range singleTables {
			tableMetadata := schema.GetTable(ref.Table)
			if tableMetadata == nil {
				continue
			}
			if tableMetadata.GetColumn(col) != nil {
				result[aliasOrName] = true
				resolved = true
			}
		}
		if !resolved {
			for t := range singleTables {
				result[t] = true
			}
			return result
		}
	}
	return result
}

func firstKey(m map[string]*TableReference) string {
	for k := range m {
		return k
	}
	return ""
}

// collectCTENames returns the set of CTE (WITH clause) names that precede the
// statement. omni does not attach UPDATE/DELETE CTEs to the statement node, so
// we parse the text before the statement Loc as a WITH clause.
func collectCTENames(fullSQL string, stmtLoc ast.Loc) map[string]bool {
	result := make(map[string]bool)
	for _, cte := range parseCTEPrefix(fullSQL, stmtLoc) {
		result[cte.Name] = true
	}
	return result
}

// writeCTEPrefix writes the statement's WITH (CTE) clause before the backup
// SELECT, so a SELECT whose FROM references a CTE stays valid. omni does not
// attach UPDATE/DELETE CTEs to the statement node, so the prefix is recovered
// from the text before the statement Loc.
func writeCTEPrefix(buf *strings.Builder, node ast.Node, fullSQL string) error {
	cte := extractCTE(fullSQL, nodeStmtLoc(node))
	if cte == "" {
		return nil
	}
	_, err := fmt.Fprintf(buf, "%s ", cte)
	return err
}

// extractCTE returns the WITH-clause text preceding the statement, or "".
func extractCTE(fullSQL string, stmtLoc ast.Loc) string {
	if stmtLoc.Start <= 0 || len(parseCTEPrefix(fullSQL, stmtLoc)) == 0 {
		return ""
	}
	return strings.TrimSpace(fullSQL[:stmtLoc.Start])
}

// nodeStmtLoc returns the source Loc of a DML statement node.
func nodeStmtLoc(node ast.Node) ast.Loc {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		return n.Loc
	case *ast.DeleteStmt:
		return n.Loc
	}
	return ast.Loc{}
}

func parseCTEPrefix(fullSQL string, stmtLoc ast.Loc) []*ast.CommonTableExpr {
	if stmtLoc.Start <= 0 || stmtLoc.Start > len(fullSQL) {
		return nil
	}
	prefix := strings.TrimSpace(fullSQL[:stmtLoc.Start])
	if prefix == "" {
		return nil
	}
	// Wrap the prefix with a dummy SELECT so it parses as a complete statement.
	parsed, err := ParseTiDBOmni(prefix + " SELECT 1")
	if err != nil || parsed == nil {
		return nil
	}
	for _, item := range parsed.Items {
		if sel, ok := item.(*ast.SelectStmt); ok && len(sel.CTEs) > 0 {
			return sel.CTEs
		}
	}
	return nil
}

func extractStatementText(fullSQL string, loc ast.Loc) string {
	start := max(0, loc.Start)
	end := loc.End
	if end <= start || end > len(fullSQL) {
		end = len(fullSQL)
	}
	return strings.TrimSpace(fullSQL[start:end])
}

func generateSQL(ctx context.Context, tCtx base.TransformContext, statementInfoList []statementInfo, databaseName string, tablePrefix string) ([]base.BackupStatement, error) {
	if len(statementInfoList) <= maxMixedDMLCount {
		return generateSQLForMixedDML(ctx, tCtx, statementInfoList, databaseName, tablePrefix)
	}
	return generateSQLForSingleTable(ctx, tCtx, statementInfoList, databaseName, tablePrefix)
}

func generateSQLForSingleTable(ctx context.Context, tCtx base.TransformContext, statementInfoList []statementInfo, databaseName string, tablePrefix string) ([]base.BackupStatement, error) {
	table := statementInfoList[0].table

	for _, item := range statementInfoList {
		if !equalTable(table, item.table) {
			return nil, errors.Errorf("prior backup cannot handle statements on different tables more than %d", maxMixedDMLCount)
		}
		// This path UNION ALLs the statements into one backup SELECT. A WITH
		// (CTE) prefix can't be emitted per union arm (TiDB rejects WITH after
		// UNION ALL), so reject CTE statements here. CTEs are still supported on
		// the per-statement mixed-DML path (<= maxMixedDMLCount).
		if extractCTE(item.fullSQL, nodeStmtLoc(item.node)) != "" {
			return nil, errors.Errorf("prior backup does not support WITH (CTE) statements when more than %d DML statements target the same table", maxMixedDMLCount)
		}
	}
	generatedColumns, normalColumns, err := classifyColumns(ctx, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc, tCtx.IsCaseSensitive, tCtx.InstanceID, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to classify columns")
	}

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Database)
	targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
	var buf strings.Builder
	if _, err := fmt.Fprintf(&buf, "CREATE TABLE `%s`.`%s` LIKE `%s`.`%s`;\n", databaseName, targetTable, table.Database, table.Table); err != nil {
		return nil, errors.Wrap(err, "failed to write create table statement")
	}

	if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s`", databaseName, targetTable); err != nil {
		return nil, errors.Wrap(err, "failed to write insert into statement")
	}
	if len(generatedColumns) > 0 {
		if _, err := buf.WriteString(" ("); err != nil {
			return nil, errors.Wrap(err, "failed to write insert into statement")
		}
		for i, column := range normalColumns {
			if i > 0 {
				if err := buf.WriteByte(','); err != nil {
					return nil, errors.Wrap(err, "failed to write comma")
				}
			}
			if _, err := fmt.Fprintf(&buf, "`%s`", column); err != nil {
				return nil, errors.Wrap(err, "failed to write column")
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return nil, errors.Wrap(err, "failed to write insert into statement")
		}
	}
	for i, item := range statementInfoList {
		if i != 0 {
			if _, err := buf.WriteString("\n  UNION ALL\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write union all statement")
			}
		}
		tableNameOrAlias := item.table.Table
		if len(item.table.Alias) > 0 {
			tableNameOrAlias = item.table.Alias
		}
		if len(generatedColumns) == 0 {
			if _, err := fmt.Fprintf(&buf, "  SELECT `%s`.* FROM ", tableNameOrAlias); err != nil {
				return nil, errors.Wrap(err, "failed to write select statement")
			}
		} else {
			if _, err := buf.WriteString("  SELECT "); err != nil {
				return nil, errors.Wrap(err, "failed to write select statement")
			}
			for j, column := range normalColumns {
				if j > 0 {
					if err := buf.WriteByte(','); err != nil {
						return nil, errors.Wrap(err, "failed to write comma")
					}
				}
				if _, err := fmt.Fprintf(&buf, "`%s`.`%s`", tableNameOrAlias, column); err != nil {
					return nil, errors.Wrap(err, "failed to write column")
				}
			}
			if _, err := buf.WriteString(" FROM "); err != nil {
				return nil, errors.Wrap(err, "failed to write from")
			}
		}
		if err := writeSuffixSelectClause(&buf, item.node, item.fullSQL); err != nil {
			return nil, errors.Wrap(err, "failed to extract suffix select statement")
		}
	}

	if err := buf.WriteByte(';'); err != nil {
		return nil, errors.Wrap(err, "failed to write semicolon")
	}

	return []base.BackupStatement{
		{
			Statement:       buf.String(),
			SourceTableName: table.Table,
			TargetTableName: targetTable,
			StartPosition:   statementInfoList[0].startPosition,
			EndPosition:     statementInfoList[len(statementInfoList)-1].endPosition,
		},
	}, nil
}

func equalTable(a, b *TableReference) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Database != "" && b.Database != "" && a.Database != b.Database {
		return false
	}
	return a.Table == b.Table
}

func generateSQLForMixedDML(ctx context.Context, tCtx base.TransformContext, statementInfoList []statementInfo, databaseName string, tablePrefix string) ([]base.BackupStatement, error) {
	var result []base.BackupStatement
	offsetLength := 1
	if len(statementInfoList) > 1 {
		offsetLength = base.GetOffsetLength(statementInfoList[len(statementInfoList)-1].offset)
	}

	for _, statementInfo := range statementInfoList {
		table := statementInfo.table
		targetTable := fmt.Sprintf("%s_%0*d_%s", tablePrefix, offsetLength, statementInfo.offset, table.Table)
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
		// If enforce_gtid_consistency = true, we cannot run CREATE TABLE .. AS SELECT.
		// So we create the table first and then run INSERT INTO .. SELECT.
		var buf strings.Builder
		if _, err := fmt.Fprintf(&buf, "CREATE TABLE `%s`.`%s` LIKE `%s`.`%s`;\n", databaseName, targetTable, table.Database, table.Table); err != nil {
			return nil, errors.Wrap(err, "failed to write create table statement")
		}
		generatedColumns, normalColumns, err := classifyColumns(ctx, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc, tCtx.IsCaseSensitive, tCtx.InstanceID, table)
		if err != nil {
			return nil, errors.Wrap(err, "failed to classify columns")
		}
		tableNameOrAlias := table.Table
		if len(table.Alias) > 0 {
			tableNameOrAlias = table.Alias
		}
		if len(generatedColumns) == 0 {
			if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` ", databaseName, targetTable); err != nil {
				return nil, errors.Wrap(err, "failed to write insert into statement")
			}
			if err := writeCTEPrefix(&buf, statementInfo.node, statementInfo.fullSQL); err != nil {
				return nil, errors.Wrap(err, "failed to write cte")
			}
			if _, err := fmt.Fprintf(&buf, "SELECT `%s`.* FROM ", tableNameOrAlias); err != nil {
				return nil, errors.Wrap(err, "failed to write select statement")
			}
		} else {
			if _, err := fmt.Fprintf(&buf, "INSERT INTO `%s`.`%s` (", databaseName, targetTable); err != nil {
				return nil, errors.Wrap(err, "failed to write insert into statement")
			}
			for i, column := range normalColumns {
				if i > 0 {
					if err := buf.WriteByte(','); err != nil {
						return nil, errors.Wrap(err, "failed to write comma")
					}
				}
				if _, err := fmt.Fprintf(&buf, "`%s`", column); err != nil {
					return nil, errors.Wrap(err, "failed to write column")
				}
			}
			if _, err := buf.WriteString(") "); err != nil {
				return nil, errors.Wrap(err, "failed to write select")
			}
			if err := writeCTEPrefix(&buf, statementInfo.node, statementInfo.fullSQL); err != nil {
				return nil, errors.Wrap(err, "failed to write cte")
			}
			if _, err := buf.WriteString("SELECT "); err != nil {
				return nil, errors.Wrap(err, "failed to write select")
			}
			for i, column := range normalColumns {
				if i > 0 {
					if err := buf.WriteByte(','); err != nil {
						return nil, errors.Wrap(err, "failed to write comma")
					}
				}
				if _, err := fmt.Fprintf(&buf, "`%s`.`%s`", tableNameOrAlias, column); err != nil {
					return nil, errors.Wrap(err, "failed to write column")
				}
			}
			if _, err := buf.WriteString(" FROM "); err != nil {
				return nil, errors.Wrap(err, "failed to write from")
			}
		}
		if err := writeSuffixSelectClause(&buf, statementInfo.node, statementInfo.fullSQL); err != nil {
			return nil, errors.Wrap(err, "failed to extract suffix select statement")
		}
		if err := buf.WriteByte(';'); err != nil {
			return nil, errors.Wrap(err, "failed to write semicolon")
		}
		result = append(result, base.BackupStatement{
			Statement:       buf.String(),
			SourceTableName: table.Table,
			TargetTableName: targetTable,
			StartPosition:   statementInfo.startPosition,
			EndPosition:     statementInfo.endPosition,
		})
	}
	return result, nil
}

// writeSuffixSelectClause writes the "<table refs> WHERE ... ORDER BY ... LIMIT ..."
// suffix of a DML statement, sliced from the original SQL via the node's Loc.
func writeSuffixSelectClause(buf *strings.Builder, node ast.Node, fullSQL string) error {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		return writeUpdateSuffix(buf, n, fullSQL)
	case *ast.DeleteStmt:
		return writeDeleteSuffix(buf, n, fullSQL)
	}
	return nil
}

func writeUpdateSuffix(buf *strings.Builder, n *ast.UpdateStmt, sql string) error {
	// "table_refs": from the first table ref to the last table ref's end.
	if len(n.Tables) > 0 {
		start := nodeLocStart(n.Tables[0])
		end := nodeLocEnd(n.Tables[len(n.Tables)-1])
		if start >= 0 && end > start && end <= len(sql) {
			if _, err := buf.WriteString(strings.TrimSpace(sql[start:end])); err != nil {
				return err
			}
		}
	}

	// Everything after the last SET assignment is WHERE + ORDER BY + LIMIT.
	if len(n.SetList) > 0 {
		lastAssign := n.SetList[len(n.SetList)-1]
		afterSet := lastAssign.Loc.End
		stmtEnd := n.Loc.End
		if stmtEnd <= 0 || stmtEnd > len(sql) {
			stmtEnd = len(sql)
		}
		if afterSet >= 0 && afterSet < stmtEnd {
			suffix := strings.TrimSpace(sql[afterSet:stmtEnd])
			if suffix != "" {
				if _, err := fmt.Fprintf(buf, " %s", suffix); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeDeleteSuffix(buf *strings.Builder, n *ast.DeleteStmt, sql string) error {
	// Single-table DELETE: from the table ref to statement end.
	// Multi-table DELETE: from the USING/reference table list to statement end.
	var start int
	switch {
	case len(n.Using) > 0:
		start = nodeLocStart(n.Using[0])
	case len(n.Tables) > 0:
		start = nodeLocStart(n.Tables[0])
	default:
		return nil
	}
	end := n.Loc.End
	if end <= 0 || end > len(sql) {
		end = len(sql)
	}
	if start >= 0 && end > start {
		if _, err := buf.WriteString(strings.TrimSpace(sql[start:end])); err != nil {
			return err
		}
	}
	return nil
}

func nodeLocStart(expr ast.TableExpr) int {
	switch e := expr.(type) {
	case *ast.TableRef:
		return e.Loc.Start
	case *ast.JoinClause:
		return e.Loc.Start
	case *ast.SubqueryExpr:
		return e.Loc.Start
	case *ast.JsonTableExpr:
		return e.Loc.Start
	}
	return -1
}

func nodeLocEnd(expr ast.TableExpr) int {
	switch e := expr.(type) {
	case *ast.TableRef:
		return e.Loc.End
	case *ast.JoinClause:
		return e.Loc.End
	case *ast.SubqueryExpr:
		return e.Loc.End
	case *ast.JsonTableExpr:
		return e.Loc.End
	}
	return -1
}

func classifyColumns(ctx context.Context, getDatabaseMetadataFunc base.GetDatabaseMetadataFunc, listDatabaseNamesFunc base.ListDatabaseNamesFunc, isCaseSensitive bool, instanceID string, table *TableReference) ([]string, []string, error) {
	if getDatabaseMetadataFunc == nil {
		return nil, nil, errors.New("GetDatabaseMetadataFunc is not set")
	}

	var dbMetadata *model.DatabaseMetadata
	allDatabaseNames, err := listDatabaseNamesFunc(ctx, instanceID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to list databases names")
	}
	if !isCaseSensitive {
		for _, db := range allDatabaseNames {
			if strings.EqualFold(db, table.Database) {
				_, dbMetadata, err = getDatabaseMetadataFunc(ctx, instanceID, db)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	} else {
		for _, db := range allDatabaseNames {
			if db == table.Database {
				_, dbMetadata, err = getDatabaseMetadataFunc(ctx, instanceID, db)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "failed to get database metadata for database %q", db)
				}
				break
			}
		}
	}
	if dbMetadata == nil {
		slog.Debug("failed to get database metadata", slog.String("instanceID", instanceID), slog.String("database", table.Database))
		return nil, nil, errors.Errorf("failed to get database metadata for InstanceID %q, Database %q", instanceID, table.Database)
	}

	schema := dbMetadata.GetSchemaMetadata("")
	if schema == nil {
		return nil, nil, errors.New("failed to get schema metadata")
	}

	tableSchema := schema.GetTable(table.Table)
	if tableSchema == nil {
		return nil, nil, errors.Errorf("failed to get table metadata for table %q", table.Table)
	}

	var generatedColumns, normalColumns []string
	for _, column := range tableSchema.GetProto().GetColumns() {
		if column.GetGeneration() != nil {
			generatedColumns = append(generatedColumns, column.GetName())
		} else {
			normalColumns = append(normalColumns, column.GetName())
		}
	}

	return generatedColumns, normalColumns, nil
}
