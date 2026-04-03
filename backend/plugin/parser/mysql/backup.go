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
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterTransformDMLToSelect(store.Engine_MYSQL, TransformDMLToSelect)
}

const (
	maxTableNameLength = 64
)

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
	StartPosition *store.Position
	EndPosition   *store.Position
	FullSQL       string
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
		omniList, err := ParseMySQLOmni(item.Text)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse sql")
		}

		for _, node := range omniList.Items {
			tables, err := ExtractTables(databaseName, node, item.Text, i, dbMetadata)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract tables")
			}
			for _, table := range tables {
				result = append(result, StatementInfo{
					Offset:        i,
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
		cteString := extractCTE(item.Node, item.FullSQL)
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
		if err := extractSuffixSelectStatement(item.Node, item.FullSQL, &buf); err != nil {
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

func extractCTE(node ast.Node, fullSQL string) string {
	var stmtStart int
	switch n := node.(type) {
	case *ast.UpdateStmt:
		stmtStart = n.Loc.Start
	case *ast.DeleteStmt:
		stmtStart = n.Loc.Start
	default:
		return ""
	}
	prefix := strings.TrimSpace(fullSQL[:stmtStart])
	if prefix == "" {
		return ""
	}
	return prefix
}

func extractCTENames(fullSQL string, stmtStart int) map[string]bool {
	prefix := strings.TrimSpace(fullSQL[:stmtStart])
	if prefix == "" {
		return nil
	}
	// Parse as "WITH ... SELECT 1" to extract CTE names.
	testSQL := prefix + " SELECT 1"
	list, err := ParseMySQLOmni(testSQL)
	if err != nil {
		return nil
	}
	result := make(map[string]bool)
	for _, item := range list.Items {
		if sel, ok := item.(*ast.SelectStmt); ok {
			for _, cte := range sel.CTEs {
				result[cte.Name] = true
			}
		}
	}
	return result
}

func extractNodeText(loc ast.Loc, sql string) string {
	if loc.Start < 0 || loc.End <= loc.Start || loc.End > len(sql) {
		return ""
	}
	return strings.TrimSpace(sql[loc.Start:loc.End])
}

func tableExprLoc(te ast.TableExpr) ast.Loc {
	switch t := te.(type) {
	case *ast.TableRef:
		return t.Loc
	case *ast.JoinClause:
		return t.Loc
	default:
		return ast.Loc{}
	}
}

func tableExprsLoc(tables []ast.TableExpr) ast.Loc {
	if len(tables) == 0 {
		return ast.Loc{}
	}
	first := tableExprLoc(tables[0])
	last := tableExprLoc(tables[len(tables)-1])
	return ast.Loc{Start: first.Start, End: last.End}
}

func extractSuffixSelectStatement(node ast.Node, fullSQL string, buf *strings.Builder) error {
	switch n := node.(type) {
	case *ast.DeleteStmt:
		return writeDeleteSuffix(buf, n, fullSQL)
	case *ast.UpdateStmt:
		return writeUpdateSuffix(buf, n, fullSQL)
	}
	return nil
}

func writeDeleteSuffix(buf *strings.Builder, n *ast.DeleteStmt, sql string) error {
	if len(n.Using) > 0 {
		// Multi-table delete: suffix starts at the USING (FROM) table references.
		usingLoc := tableExprsLoc(n.Using)
		suffix := strings.TrimSpace(sql[usingLoc.Start:n.Loc.End])
		_, err := buf.WriteString(suffix)
		return err
	}

	// Single-table delete: suffix from table ref to end of statement.
	if len(n.Tables) > 0 {
		tablesLoc := tableExprsLoc(n.Tables)
		suffix := strings.TrimSpace(sql[tablesLoc.Start:n.Loc.End])
		_, err := buf.WriteString(suffix)
		return err
	}

	return nil
}

func writeUpdateSuffix(buf *strings.Builder, n *ast.UpdateStmt, sql string) error {
	// Table references text.
	tablesLoc := tableExprsLoc(n.Tables)
	tablesText := extractNodeText(tablesLoc, sql)
	if _, err := buf.WriteString(tablesText); err != nil {
		return err
	}

	// Everything after the last SET assignment (WHERE, ORDER BY, LIMIT).
	if len(n.SetList) > 0 {
		lastSet := n.SetList[len(n.SetList)-1]
		rest := strings.TrimSpace(sql[lastSet.Loc.End:n.Loc.End])
		if rest != "" {
			if err := buf.WriteByte(' '); err != nil {
				return err
			}
			if _, err := buf.WriteString(rest); err != nil {
				return err
			}
		}
	}

	return nil
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

	emptySchema := ""
	schema := dbMetadata.GetSchemaMetadata(emptySchema)
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

// ExtractTables extracts table references from a DML statement node.
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
	cteMap := extractCTENames(fullSQL, n.Loc.Start)

	if len(n.Using) == 0 {
		// Single-table delete.
		if len(n.Tables) == 0 {
			return nil, nil
		}
		ref, ok := n.Tables[0].(*ast.TableRef)
		if !ok {
			return nil, nil
		}

		database := ref.Schema
		table := ref.Name
		if database != "" && database != databaseName {
			return nil, errors.Errorf("database is not matched: %s != %s", database, databaseName)
		}
		if database == "" {
			database = databaseName
		}

		return []StatementInfo{{
			Offset:    offset,
			Statement: fullSQL,
			Node:      n,
			Table: &TableReference{
				Database:      database,
				Table:         table,
				Alias:         ref.Alias,
				StatementType: StatementTypeDelete,
			},
			FullSQL: fullSQL,
		}}, nil
	}

	// Multi-table delete.
	singleTables := collectSingleTables(databaseName, n.Using)

	var result []StatementInfo
	for _, te := range n.Tables {
		ref, ok := te.(*ast.TableRef)
		if !ok {
			continue
		}

		database := ref.Schema
		table := ref.Name
		if database != "" && database != databaseName {
			return nil, errors.Errorf("database is not matched: %s != %s", database, databaseName)
		}

		if database == "" && cteMap[table] {
			continue
		}

		singleTable, ok := singleTables[table]
		if !ok {
			return nil, errors.Errorf("cannot extract reference table: no matched table %q in referenced table list", table)
		}

		singleTable.StatementType = StatementTypeDelete
		result = append(result, StatementInfo{
			Offset:    offset,
			Statement: fullSQL,
			Node:      n,
			Table:     singleTable,
			FullSQL:   fullSQL,
		})
	}

	return result, nil
}

func extractTablesFromUpdate(databaseName string, n *ast.UpdateStmt, fullSQL string, offset int, dbMetadata *model.DatabaseMetadata) ([]StatementInfo, error) {
	cteMap := extractCTENames(fullSQL, n.Loc.Start)

	// Collect tables being updated from the SET clause.
	updatedTables := make(map[string]bool)
	var unqualifiedColumns []string
	for _, assignment := range n.SetList {
		table := assignment.Column.Table
		if _, isCTE := cteMap[table]; !isCTE {
			updatedTables[table] = true
			if table == "" {
				unqualifiedColumns = append(unqualifiedColumns, assignment.Column.Column)
			}
		}
	}

	singleTables := collectSingleTables(databaseName, n.Tables)

	// Resolve unqualified columns using metadata.
	if _, exists := updatedTables[""]; exists {
		delete(updatedTables, "")
		resolved := resolveUnqualifiedColumns(unqualifiedColumns, singleTables, dbMetadata)
		for tableName := range resolved {
			updatedTables[tableName] = true
		}
	}

	var result []StatementInfo
	for table := range updatedTables {
		singleTable, ok := singleTables[table]
		if !ok {
			return nil, errors.Errorf("cannot extract reference table: no matched updated table %q in referenced table list", table)
		}

		singleTable.StatementType = StatementTypeUpdate
		result = append(result, StatementInfo{
			Offset:    offset,
			Statement: fullSQL,
			Node:      n,
			Table:     singleTable,
			FullSQL:   fullSQL,
		})
	}

	return result, nil
}

// collectSingleTables walks table expressions and returns a map from
// table name (or alias) to TableReference.
func collectSingleTables(databaseName string, tables []ast.TableExpr) map[string]*TableReference {
	result := make(map[string]*TableReference)
	for _, te := range tables {
		collectSingleTablesFromExpr(databaseName, te, result)
	}
	return result
}

func collectSingleTablesFromExpr(databaseName string, te ast.TableExpr, result map[string]*TableReference) {
	switch t := te.(type) {
	case *ast.TableRef:
		database := t.Schema
		if database == "" {
			database = databaseName
		}
		ref := &TableReference{
			Database: database,
			Table:    t.Name,
		}
		if t.Alias != "" {
			ref.Alias = t.Alias
			result[t.Alias] = ref
		} else {
			result[t.Name] = ref
		}
	case *ast.JoinClause:
		collectSingleTablesFromExpr(databaseName, t.Left, result)
		collectSingleTablesFromExpr(databaseName, t.Right, result)
	}
}

// resolveUnqualifiedColumns resolves unqualified column names to their owning
// table(s) using database metadata. Falls back to all tables if metadata is
// unavailable or column cannot be resolved.
func resolveUnqualifiedColumns(columns []string, singleTables map[string]*TableReference, dbMetadata *model.DatabaseMetadata) map[string]bool {
	result := make(map[string]bool)

	var schema *model.SchemaMetadata
	if dbMetadata != nil {
		schema = dbMetadata.GetSchemaMetadata("")
	}
	if schema == nil {
		for tableName := range singleTables {
			result[tableName] = true
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
			for tableName := range singleTables {
				result[tableName] = true
			}
			return result
		}
	}

	return result
}
