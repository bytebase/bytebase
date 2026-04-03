package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MYSQL, TransformDMLToSelect)
}

const (
	maxTableNameLength = 64
)

type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Database      string
	Table         string
	Alias         string
	StatementType StatementType
}

type StatementInfo struct {
	Offset        int
	Statement     string
	Node          ast.Node
	Table         *TableReference
	StartPosition *storepb.Position
	EndPosition   *storepb.Position
	FullSQL       string
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

func prepareTransformation(databaseName, statement string, dbMetadata *model.DatabaseMetadata) ([]StatementInfo, error) {
	list, err := SplitSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split sql")
	}

	var result []StatementInfo
	for i, item := range list {
		if len(item.Text) == 0 || item.Empty {
			continue
		}

		parsed, err := ParseMySQLOmni(item.Text)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse sql")
		}
		if parsed == nil || len(parsed.Items) == 0 {
			continue
		}

		for _, node := range parsed.Items {
			tables, err := ExtractTables(databaseName, node, item.Text, i, dbMetadata)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract tables")
			}
			for _, table := range tables {
				result = append(result, StatementInfo{
					Offset:        table.Offset,
					Statement:     table.Statement,
					Node:          table.Node,
					Table:         table.Table,
					StartPosition: item.Start,
					EndPosition:   item.End,
					FullSQL:       item.Text,
				})
			}
		}
	}

	return result, nil
}

// ExtractTables extracts table references from a DML statement.
func ExtractTables(databaseName string, node ast.Node, fullSQL string, offset int, dbMetadata *model.DatabaseMetadata) ([]StatementInfo, error) {
	switch n := node.(type) {
	case *ast.DeleteStmt:
		return extractTablesFromDelete(databaseName, n, fullSQL, offset, dbMetadata)
	case *ast.UpdateStmt:
		return extractTablesFromUpdate(databaseName, n, fullSQL, offset, dbMetadata)
	default:
		return nil, nil
	}
}

func extractTablesFromDelete(databaseName string, n *ast.DeleteStmt, fullSQL string, offset int, _ *model.DatabaseMetadata) ([]StatementInfo, error) {
	cteNames := collectCTENames(fullSQL, n.Loc)
	stmtText := extractStatementText(fullSQL, n.Loc)

	// Collect all single table refs from the Tables list.
	singleTables := collectSingleTables(databaseName, n.Tables)

	if len(n.Using) == 0 {
		// Single-table DELETE: DELETE FROM t WHERE ...
		if len(singleTables) == 0 {
			return nil, nil
		}
		for _, ref := range singleTables {
			ref.StatementType = StatementTypeDelete
		}
		first := singleTables[firstKey(singleTables)]
		return []StatementInfo{{
			Offset:    offset,
			Statement: stmtText,
			Node:      n,
			Table:     first,
			FullSQL:   fullSQL,
		}}, nil
	}

	// Multi-table DELETE: DELETE t1, t2 FROM t1 JOIN t2 ... WHERE ...
	// Tables = targets to delete from, Using = referenced table list.
	refTables := collectSingleTables(databaseName, n.Using)

	var result []StatementInfo
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
		ref.StatementType = StatementTypeDelete
		result = append(result, StatementInfo{
			Offset:    offset,
			Statement: stmtText,
			Node:      n,
			Table:     ref,
			FullSQL:   fullSQL,
		})
	}
	return result, nil
}

func extractTablesFromUpdate(databaseName string, n *ast.UpdateStmt, fullSQL string, offset int, dbMetadata *model.DatabaseMetadata) ([]StatementInfo, error) {
	cteNames := collectCTENames(fullSQL, n.Loc)
	stmtText := extractStatementText(fullSQL, n.Loc)
	singleTables := collectSingleTables(databaseName, n.Tables)

	// Determine which tables are updated via SET clause column prefixes.
	updatedTables := make(map[string]bool)
	var unqualifiedColumns []string
	for _, assign := range n.SetList {
		if assign.Column != nil {
			table := assign.Column.Table
			if !cteNames[table] {
				updatedTables[table] = true
				if table == "" {
					unqualifiedColumns = append(unqualifiedColumns, assign.Column.Column)
				}
			}
		}
	}

	// Resolve unqualified columns using metadata.
	if updatedTables[""] {
		delete(updatedTables, "")
		resolved := resolveUnqualifiedColumns(unqualifiedColumns, singleTables, dbMetadata)
		for t := range resolved {
			updatedTables[t] = true
		}
	}

	// Single-table UPDATE (only one table, no explicit qualification needed).
	if len(updatedTables) == 0 && len(singleTables) == 1 {
		for _, ref := range singleTables {
			ref.StatementType = StatementTypeUpdate
			return []StatementInfo{{
				Offset:    offset,
				Statement: stmtText,
				Node:      n,
				Table:     ref,
				FullSQL:   fullSQL,
			}}, nil
		}
	}

	var result []StatementInfo
	for table := range updatedTables {
		ref, ok := singleTables[table]
		if !ok {
			return nil, errors.Errorf("cannot extract reference table: no matched updated table %q in referenced table list", table)
		}
		ref.StatementType = StatementTypeUpdate
		result = append(result, StatementInfo{
			Offset:    offset,
			Statement: stmtText,
			Node:      n,
			Table:     ref,
			FullSQL:   fullSQL,
		})
	}
	return result, nil
}

// collectSingleTables walks TableExpr slices and collects TableRef entries keyed by alias or name.
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
		ref := &TableReference{
			Database: db,
			Table:    e.Name,
			Alias:    e.Alias,
		}
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

// collectCTENames extracts CTE names from the SQL text before the statement Loc.
// MySQL's UPDATE/DELETE CTEs are not in the omni AST, so we parse the prefix.
func collectCTENames(fullSQL string, stmtLoc ast.Loc) map[string]bool {
	result := make(map[string]bool)
	if stmtLoc.Start <= 0 {
		return result
	}
	prefix := strings.TrimSpace(fullSQL[:stmtLoc.Start])
	if !strings.HasPrefix(strings.ToUpper(prefix), "WITH") {
		return result
	}
	// Parse "prefix SELECT 1" to extract CTE names via omni.
	tempSQL := prefix + " SELECT 1"
	parsed, err := ParseMySQLOmni(tempSQL)
	if err != nil || parsed == nil {
		return result
	}
	for _, item := range parsed.Items {
		if sel, ok := item.(*ast.SelectStmt); ok {
			for _, cte := range sel.CTEs {
				result[cte.Name] = true
			}
		}
	}
	return result
}

// extractCTE extracts the CTE (WITH clause) text from the SQL before the statement.
func extractCTE(fullSQL string, stmtLoc ast.Loc) string {
	if stmtLoc.Start <= 0 {
		return ""
	}
	prefix := strings.TrimSpace(fullSQL[:stmtLoc.Start])
	if !strings.HasPrefix(strings.ToUpper(prefix), "WITH") {
		return ""
	}
	return prefix
}

func extractStatementText(fullSQL string, loc ast.Loc) string {
	end := loc.End
	if end <= 0 || end > len(fullSQL) {
		end = len(fullSQL)
	}
	start := 0
	if loc.Start > 0 {
		start = loc.Start
	}
	return strings.TrimSpace(fullSQL[start:end])
}

// writeSuffixSelectClause writes the FROM ... WHERE ... suffix of a DML statement.
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
	// For UPDATE: write "table_refs WHERE ... ORDER BY ... LIMIT ..."
	// Use the last table's Loc.End as the boundary — everything from first table
	// to last table's end is the table reference list.
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
		suffix := strings.TrimSpace(sql[afterSet:stmtEnd])
		if suffix != "" {
			if _, err := fmt.Fprintf(buf, " %s", suffix); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeDeleteSuffix(buf *strings.Builder, n *ast.DeleteStmt, sql string) error {
	// For single-table DELETE: everything from the table ref to statement end.
	// For multi-table DELETE: everything from the USING/reference table list to end.
	if len(n.Using) > 0 {
		start := nodeLocStart(n.Using[0])
		end := n.Loc.End
		if end <= 0 || end > len(sql) {
			end = len(sql)
		}
		if start >= 0 && end > start {
			if _, err := buf.WriteString(strings.TrimSpace(sql[start:end])); err != nil {
				return err
			}
		}
	} else if len(n.Tables) > 0 {
		start := nodeLocStart(n.Tables[0])
		end := n.Loc.End
		if end <= 0 || end > len(sql) {
			end = len(sql)
		}
		if start >= 0 && end > start {
			if _, err := buf.WriteString(strings.TrimSpace(sql[start:end])); err != nil {
				return err
			}
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
	}
	return -1
}

func nodeLocEnd(expr ast.TableExpr) int {
	switch e := expr.(type) {
	case *ast.TableRef:
		return e.Loc.End
	case *ast.JoinClause:
		return e.Loc.End
	}
	return -1
}

// resolveUnqualifiedColumns resolves unqualified column names to owning tables using metadata.
func resolveUnqualifiedColumns(columns []string, singleTables map[string]*TableReference, dbMetadata *model.DatabaseMetadata) map[string]bool {
	result := make(map[string]bool)

	if dbMetadata == nil {
		for t := range singleTables {
			result[t] = true
		}
		return result
	}
	schema := dbMetadata.GetSchemaMetadata("")
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

func generateSQL(ctx context.Context, tCtx base.TransformContext, statementInfoList []StatementInfo, databaseName string, tablePrefix string) ([]base.BackupStatement, error) {
	groupByTable := make(map[string][]StatementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.Table.Database, item.Table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	for key, list := range groupByTable {
		stmtType := StatementTypeUnknown
		for _, item := range list {
			if stmtType == StatementTypeUnknown {
				stmtType = item.Table.StatementType
			}
			if stmtType != item.Table.StatementType {
				return nil, errors.Errorf("prior backup cannot handle mixed DML statements on the same table %s", key)
			}
		}
	}

	var result []base.BackupStatement
	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(ctx, tCtx, list, databaseName, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table %s", key)
		}
		result = append(result, *backupStatement)
	}

	slices.SortFunc(result, func(i, j base.BackupStatement) int {
		if i.StartPosition.Line != j.StartPosition.Line {
			if i.StartPosition.Line < j.StartPosition.Line {
				return -1
			}
			return 1
		}
		if i.StartPosition.Column != j.StartPosition.Column {
			if i.StartPosition.Column < j.StartPosition.Column {
				return -1
			}
			return 1
		}
		if i.SourceTableName < j.SourceTableName {
			return -1
		}
		if i.SourceTableName > j.SourceTableName {
			return 1
		}
		return 0
	})

	return result, nil
}

func generateSQLForTable(ctx context.Context, tCtx base.TransformContext, statementInfoList []StatementInfo, databaseName string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].Table

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
			if _, err := buf.WriteString("\n  UNION DISTINCT\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write union all statement")
			}
		}
		tableNameOrAlias := item.Table.Table
		if len(item.Table.Alias) > 0 {
			tableNameOrAlias = item.Table.Alias
		}
		if _, err := buf.WriteString("  "); err != nil {
			return nil, errors.Wrap(err, "failed to write space")
		}

		stmtLoc := nodeStmtLoc(item.Node)
		cteString := extractCTE(item.FullSQL, stmtLoc)
		if len(cteString) > 0 {
			if _, err := fmt.Fprintf(&buf, "%s ", cteString); err != nil {
				return nil, errors.Wrap(err, "failed to write cte")
			}
		}

		if len(generatedColumns) == 0 {
			if _, err := fmt.Fprintf(&buf, "SELECT `%s`.* FROM ", tableNameOrAlias); err != nil {
				return nil, errors.Wrap(err, "failed to write select statement")
			}
		} else {
			if _, err := buf.WriteString("SELECT "); err != nil {
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
		if err := writeSuffixSelectClause(&buf, item.Node, item.FullSQL); err != nil {
			return nil, errors.Wrap(err, "failed to extract suffix select statement")
		}
	}

	if err := buf.WriteByte(';'); err != nil {
		return nil, errors.Wrap(err, "failed to write semicolon")
	}

	return &base.BackupStatement{
		Statement:       buf.String(),
		SourceTableName: table.Table,
		TargetTableName: targetTable,
		StartPosition:   statementInfoList[0].StartPosition,
		EndPosition:     statementInfoList[len(statementInfoList)-1].EndPosition,
	}, nil
}

func nodeStmtLoc(node ast.Node) ast.Loc {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		return n.Loc
	case *ast.DeleteStmt:
		return n.Loc
	}
	return ast.Loc{}
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

	var tableSchema *model.TableMetadata
	if !isCaseSensitive {
		for _, tableName := range schema.ListTableNames() {
			if strings.EqualFold(tableName, table.Table) {
				tableSchema = schema.GetTable(tableName)
				break
			}
		}
	} else {
		tableSchema = schema.GetTable(table.Table)
	}
	if tableSchema == nil {
		return nil, nil, errors.Errorf("table %s not found in schema", table.Table)
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
