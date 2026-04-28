package tsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_MSSQL, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	if len(originalSQL) == 0 {
		return "", errors.Errorf("no original SQL")
	}

	parsedStatements, err := parseTSQLStatements(statement)
	if err != nil {
		return "", err
	}

	// Find the AST that contains the statement at the backup position
	var targetResult ast.Node
	for _, parsedStatement := range parsedStatements {
		node, ok := GetOmniNode(parsedStatement.AST)
		if !ok || node == nil || !isRestorableDML(node) {
			continue
		}
		start, end := statementPositions(parsedStatement.Start, parsedStatement.Text, dmlNodeLoc(node))
		if inRange(start, end, backupItem.StartPosition, backupItem.EndPosition) {
			targetResult = node
			break
		}
	}

	if targetResult == nil {
		return "", errors.Errorf("could not find statement at position (line %d:%d - %d:%d)",
			backupItem.StartPosition.Line, backupItem.StartPosition.Column,
			backupItem.EndPosition.Line, backupItem.EndPosition.Column)
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}
	return doGenerate(ctx, rCtx, sqlForComment, targetResult, backupItem)
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

	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Errorf("GetDatabaseMetadataFunc is required")
	}

	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}

	if metadata == nil {
		return "", errors.Errorf("database metadata is nil")
	}

	schema := backupItem.SourceTable.Schema
	if schema == "" {
		schema = "dbo"
	}
	schemaMetadata := metadata.GetSchemaMetadata(schema)
	if schemaMetadata == nil {
		return "", errors.Errorf("schema metadata not found for %s", schema)
	}

	tableMetadata := schemaMetadata.GetTable(backupItem.SourceTable.Table)
	if tableMetadata == nil {
		return "", errors.Errorf("table metadata not found for %s.%s", schema, backupItem.SourceTable.Table)
	}

	g := &generator{
		ctx:              ctx,
		rCtx:             rCtx,
		backupDatabase:   targetDatabase,
		backupTable:      backupItem.TargetTable.Table,
		originalDatabase: sourceDatabase,
		originalSchema:   schema,
		originalTable:    backupItem.SourceTable.Table,
		pk:               tableMetadata.GetPrimaryKey(),
		table:            tableMetadata,
	}
	result, err := g.generate(node)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, result), nil
}

type generator struct {
	ctx  context.Context
	rCtx base.RestoreContext

	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalSchema   string
	originalTable    string
	pk               *model.IndexMetadata
	table            *model.TableMetadata
}

// hasIdentityColumn checks if the table has any IDENTITY columns
func (g *generator) hasIdentityColumn() bool {
	if g.table == nil {
		return false
	}
	for _, col := range g.table.GetProto().GetColumns() {
		if col.IsIdentity {
			return true
		}
	}
	return false
}

func (g *generator) generate(node ast.Node) (string, error) {
	switch n := node.(type) {
	case *ast.DeleteStmt:
		return g.generateDelete()
	case *ast.UpdateStmt:
		return g.generateUpdate(n)
	default:
		return "", errors.Errorf("unsupported restore statement %T", node)
	}
}

func (g *generator) generateDelete() (string, error) {
	// Check if the table has IDENTITY columns
	hasIdentity := g.hasIdentityColumn()

	if hasIdentity {
		// For tables with IDENTITY columns, we need to enable IDENTITY_INSERT
		// and use explicit column lists
		var buf strings.Builder

		// Build column list
		var columnList strings.Builder
		for i, column := range g.table.GetProto().GetColumns() {
			if i > 0 {
				columnList.WriteString(", ")
			}
			fmt.Fprintf(&columnList, "[%s]", column.Name)
		}

		fmt.Fprintf(&buf, "SET IDENTITY_INSERT [%s].[%s].[%s] ON;\n",
			g.originalDatabase, g.originalSchema, g.originalTable)
		fmt.Fprintf(&buf, "INSERT INTO [%s].[%s].[%s] (%s) SELECT %s FROM [%s].[dbo].[%s];\n",
			g.originalDatabase, g.originalSchema, g.originalTable,
			columnList.String(),
			columnList.String(),
			g.backupDatabase, g.backupTable)
		fmt.Fprintf(&buf, "SET IDENTITY_INSERT [%s].[%s].[%s] OFF;\n",
			g.originalDatabase, g.originalSchema, g.originalTable)
		fmt.Fprintf(&buf, "EXEC('DBCC CHECKIDENT (''[%s].[%s].[%s]'', RESEED)');",
			g.originalDatabase, g.originalSchema, g.originalTable)

		return buf.String(), nil
	}

	// Simple INSERT for tables without IDENTITY columns
	return fmt.Sprintf(`INSERT INTO [%s].[%s].[%s] SELECT * FROM [%s].[dbo].[%s];`,
		g.originalDatabase, g.originalSchema, g.originalTable,
		g.backupDatabase, g.backupTable), nil
}

func disjoint(a []string, b map[string]bool) bool {
	for _, item := range a {
		if _, ok := b[item]; ok {
			return false
		}
	}
	return true
}

func (g *generator) findDisjointUniqueKey(updateColumns []string) ([]string, error) {
	columnMap := make(map[string]bool)
	for _, column := range updateColumns {
		columnMap[column] = true
	}
	if g.pk != nil {
		if disjoint(g.pk.GetProto().Expressions, columnMap) {
			return g.pk.GetProto().Expressions, nil
		}
	}
	for _, index := range g.table.GetProto().Indexes {
		if index.Primary {
			continue
		}
		if !index.Unique {
			continue
		}
		if disjoint(index.Expressions, columnMap) {
			return index.Expressions, nil
		}
	}
	return nil, errors.Errorf("no disjoint unique key found for %s.%s.%s", g.originalDatabase, g.originalSchema, g.originalTable)
}

func (g *generator) generateUpdate(stmt *ast.UpdateStmt) (string, error) {
	updateColumns := updateSetColumns(stmt.SetClause)
	uk, err := g.findDisjointUniqueKey(updateColumns)
	if err != nil {
		return "", err
	}

	var buf strings.Builder

	// Check if the table has IDENTITY columns
	hasIdentity := g.hasIdentityColumn()

	if hasIdentity {
		// Enable IDENTITY_INSERT for MERGE statement
		if _, err := fmt.Fprintf(&buf, "SET IDENTITY_INSERT [%s].[%s].[%s] ON;\n",
			g.originalDatabase, g.originalSchema, g.originalTable); err != nil {
			return "", err
		}
	}

	if _, err := fmt.Fprintf(&buf, "MERGE INTO [%s].[%s].[%s] AS t\nUSING [%s].[dbo].[%s] AS b\n  ON", g.originalDatabase, g.originalSchema, g.originalTable, g.backupDatabase, g.backupTable); err != nil {
		return "", err
	}
	for i, column := range uk {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, " AND"); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprintf(&buf, " t.[%s] = b.[%s]", column, column); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprint(&buf, "\nWHEN MATCHED THEN\n  UPDATE SET"); err != nil {
		return "", err
	}
	for i, field := range updateColumns {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, ","); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprintf(&buf, " t.[%s] = b.[%s]", field, field); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprint(&buf, "\nWHEN NOT MATCHED THEN\n INSERT ("); err != nil {
		return "", err
	}
	for i, column := range g.table.GetProto().GetColumns() {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, ", "); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprintf(&buf, "[%s]", column.Name); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprint(&buf, ") VALUES ("); err != nil {
		return "", err
	}
	for i, column := range g.table.GetProto().GetColumns() {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, ", "); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprintf(&buf, "b.[%s]", column.Name); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprint(&buf, ");"); err != nil {
		return "", err
	}

	// Check if we need to disable IDENTITY_INSERT and reseed
	if hasIdentity {
		if _, err := fmt.Fprintf(&buf, "\n\nSET IDENTITY_INSERT [%s].[%s].[%s] OFF;\nEXEC('DBCC CHECKIDENT (''[%s].[%s].[%s]'', RESEED)');",
			g.originalDatabase, g.originalSchema, g.originalTable,
			g.originalDatabase, g.originalSchema, g.originalTable); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

func updateSetColumns(setClause *ast.List) []string {
	if setClause == nil {
		return nil
	}
	var columns []string
	for _, item := range setClause.Items {
		setExpr, ok := item.(*ast.SetExpr)
		if !ok {
			continue
		}
		if setExpr.Column != nil {
			columns = append(columns, setExpr.Column.Column)
			continue
		}
		if setExpr.VarColumn != nil {
			columns = append(columns, setExpr.VarColumn.Column)
			continue
		}
		if setExpr.Variable != "" {
			if column := updateColumnFromDualAssignment(setExpr.Value); column != "" {
				columns = append(columns, column)
			}
		}
	}
	return columns
}

func updateColumnFromDualAssignment(expr ast.ExprNode) string {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if e.Op == ast.BinOpEq {
			return updateColumnFromExpr(e.Left)
		}
	case *ast.ParenExpr:
		return updateColumnFromDualAssignment(e.Expr)
	default:
	}
	return ""
}

func updateColumnFromExpr(expr ast.ExprNode) string {
	switch e := expr.(type) {
	case *ast.ColumnRef:
		return e.Column
	case *ast.ParenExpr:
		return updateColumnFromExpr(e.Expr)
	default:
	}
	return ""
}

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	parseResults, err := parseTSQLStatements(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse statement")
	}

	var originalSQL []string
	for _, parsedStatement := range parseResults {
		node, ok := GetOmniNode(parsedStatement.AST)
		if !ok || node == nil || !isRestorableDML(node) {
			continue
		}
		start, end := statementPositions(parsedStatement.Start, parsedStatement.Text, dmlNodeLoc(node))
		if !inRange(start, end, backupItem.StartPosition, backupItem.EndPosition) {
			continue
		}
		sql := strings.TrimSuffix(sourceFromLoc(parsedStatement.Text, dmlNodeLoc(node)), ";")
		originalSQL = append(originalSQL, sql)
	}
	if len(originalSQL) == 0 {
		return "", nil
	}

	// Join with semicolons and add a trailing semicolon
	return strings.Join(originalSQL, ";\n") + ";", nil
}

func isRestorableDML(node ast.Node) bool {
	switch node.(type) {
	case *ast.UpdateStmt, *ast.DeleteStmt:
		return true
	default:
		return false
	}
}

func statementPositions(start *storepb.Position, source string, loc ast.Loc) (*storepb.Position, *storepb.Position) {
	return positionFromByteOffset(start, source, loc.Start), positionFromByteOffset(start, source, dmlEndOffset(source, loc))
}

// inRange checks if a statement's position range is within the target range.
func inRange(start, end, targetStart, targetEnd *storepb.Position) bool {
	if start.Line < targetStart.Line || (start.Line == targetStart.Line && start.Column < targetStart.Column) {
		return false
	}
	if end.Line > targetEnd.Line || (end.Line == targetEnd.Line && end.Column > targetEnd.Column) {
		return false
	}
	return true
}
